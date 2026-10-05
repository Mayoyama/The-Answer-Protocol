//! TAP CLI client: connects to the server, logs in, and relays stdin and server lines until disconnect.
use crate::Event::{ConnErr, CtrlC, CtrlD, ServerInput, UserInput};
use anyhow::{Context, Result, bail};
use ctrlc::set_handler;
use std::sync::mpsc::{
    RecvTimeoutError::{Disconnected, Timeout},
    channel,
};
use std::{
    collections::VecDeque,
    thread,
    time::{Duration, Instant},
};
use std::{
    io::{BufRead, BufReader, Write, stdin},
    net::TcpStream,
};

/// Where the client is in the login handshake; input typed before login is queued until it completes.
#[derive(PartialEq, Clone, Copy)]
enum LoginState {
    TCPWaiting,
    ConnCommReady,
    ConnCommReplyPending,
    LoggedIn,
}

/// Messages sent to the main loop by the reader threads and the Ctrl+C handler.
enum Event {
    UserInput(String),
    ServerInput(String),
    ConnErr(String),
    CtrlC,
    CtrlD,
}

/// Connection state: the server writer, login progress, input queued before login, and a pending Ctrl+D.
struct Login {
    tap_writer: TcpStream,
    login_state: LoginState,
    stdin_gate_line: VecDeque<String>,
    ctrl_d_pending: bool,
}

impl Login {
    /// Sends CONNECT <username> and waits for the server's reply.
    fn handle_login(&mut self, input: &str) -> Result<()> {
        let trimmed_input = input.trim_end();

        writeln!(self.tap_writer, "CONNECT {}", trimmed_input)
            .context("Error writing to TAP server")?;

        self.login_state = LoginState::ConnCommReplyPending;

        Ok(())
    }

    /// Sends QUIT to the server, giving the write at most \duration`.`
    fn handle_quit(&mut self, duration: Duration) -> Result<()> {
        self.tap_writer
            .set_write_timeout(Some(duration))
            .context("Failed to set write timeout")?;

        writeln!(self.tap_writer, "QUIT").context("Error writing to TAP server")?;

        Ok(())
    }

    /// Sends all input queued during login to the server, in order.
    fn clear_stdin_gate(&mut self) -> Result<()> {
        for item in self.stdin_gate_line.drain(..) {
            writeln!(self.tap_writer, "{}", item.trim_end())
                .context("Error writing to TAP server")?;
        }

        Ok(())
    }
}

/// Runs the client and prints any error it ends with.
fn main() {
    if let Err(err) = run() {
        println!("{err:#}");
    }
}

/// Connects to the server, starts the server and stdin reader threads, and handles events until the player quits or the connection is lost.
fn run() -> Result<()> {
    let (user_input, event_recv) = channel::<Event>();
    let server_responses = user_input.clone();
    let sigint_send = user_input.clone();

    set_handler(move || {
        let _ = sigint_send.send(CtrlC);
    })
    .context("Failed to establish Ctrl+C handler")?;

    let stream = TcpStream::connect("127.0.0.1:4242").context("Could not connect to TCP: 4242")?;

    let quit_timeout_duration = Duration::from_secs(3);
    let command_timeout_duration = Duration::from_secs(5);

    let mut login = Login {
        tap_writer: stream.try_clone().context("Failed to clone stream")?,
        login_state: LoginState::TCPWaiting,
        stdin_gate_line: VecDeque::new(),
        ctrl_d_pending: false,
    };

    login
        .tap_writer
        .set_write_timeout(Some(command_timeout_duration))
        .context("Failed to set write timeout")?;

    let mut tap_reader = BufReader::new(stream);
    let mut user_read_line = String::new();
    let mut server_read_line = String::new();

    thread::spawn(move || {
        loop {
            server_read_line.clear();

            match tap_reader.read_line(&mut server_read_line) {
                Ok(0) => {
                    let _ = server_responses.send(ConnErr("Connection Lost".to_string()));

                    return;
                }
                Ok(_) => {
                    if server_responses
                        .send(ServerInput(server_read_line.clone()))
                        .is_err()
                    {
                        return;
                    }
                }
                Err(err) => {
                    let _ = server_responses.send(ConnErr(err.to_string()));

                    return;
                }
            }
        }
    });

    thread::spawn(move || {
        loop {
            user_read_line.clear();

            match stdin().read_line(&mut user_read_line) {
                Ok(0) => {
                    let _ = user_input.send(CtrlD);

                    return;
                }
                Ok(_) => {
                    if user_input.send(UserInput(user_read_line.clone())).is_err() {
                        return;
                    }
                }
                Err(err) => {
                    let _ = user_input.send(ConnErr(err.to_string()));

                    return;
                }
            }
        }
    });

    for recv in &event_recv {
        match recv {
            UserInput(input) => {
                if login.login_state == LoginState::LoggedIn {
                    writeln!(login.tap_writer, "{}", input.trim_end())
                        .context("Error writing to TAP server")?;

                    continue;
                }

                if login.login_state == LoginState::ConnCommReady {
                    login.handle_login(&input)?;
                } else {
                    login.stdin_gate_line.push_back(input);
                }
            }

            ServerInput(response) => {
                let trimmed_response = response.trim_end();

                if login.login_state == LoginState::TCPWaiting {
                    login.login_state = LoginState::ConnCommReady;

                    println!("{}", response);

                    println!("Enter a new username between 2 and 10 characters:");

                    if let Some(input_line) = login.stdin_gate_line.pop_front() {
                        login.handle_login(&input_line)?;
                    }

                    continue;
                }

                if login.login_state == LoginState::ConnCommReplyPending {
                    println!("{}", response);

                    if trimmed_response.starts_with("OK") {
                        login.login_state = LoginState::LoggedIn;

                        login.clear_stdin_gate()?;
                    } else {
                        login.login_state = LoginState::ConnCommReady;

                        println!("Enter a new username between 2 and 10 characters:");

                        if let Some(input_line) = login.stdin_gate_line.pop_front() {
                            login.handle_login(&input_line)?;
                        }
                    }

                    if login.ctrl_d_pending && login.stdin_gate_line.is_empty() {
                        login.handle_quit(quit_timeout_duration)?;

                        break;
                    }

                    continue;
                }

                println!("{}", trimmed_response);

                if response.trim() == "OK bye" {
                    return Ok(());
                }
            }

            ConnErr(err) => {
                bail!("{err}");
            }

            CtrlC => {
                println!("Ctrl+C or shutdown request detected. Closing connection to server...");
                login.handle_quit(quit_timeout_duration)?;

                break;
            }

            CtrlD => {
                println!("Ctrl + D detected. Closing connection to server...");

                if !login.stdin_gate_line.is_empty() {
                    login.ctrl_d_pending = true;
                } else {
                    login.handle_quit(quit_timeout_duration)?;

                    break;
                }
            }
        }
    }

    let deadline = Instant::now() + quit_timeout_duration;

    loop {
        let remaining = deadline.saturating_duration_since(Instant::now());

        match event_recv.recv_timeout(remaining) {
            Ok(ServerInput(msg)) => {
                if msg.starts_with("OK bye") {
                    println!("{}", msg.trim_end());

                    break;
                }
            }

            Ok(ConnErr(err)) => {
                bail!("ERROR READING_TO_FROM_SERVER: {err}.\nDISCONNECTED.");
            }

            Ok(..) => {}

            Err(Timeout) => {
                bail!("No response from server... exiting program anyway");
            }

            Err(Disconnected) => {
                bail!("ERROR READING_TO_FROM_SERVER.\nDISCONNECTED.");
            }
        }
    }

    Ok(())
}
