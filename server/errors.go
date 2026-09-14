package main

import "fmt"

// ProtocolError is a protocol-level error carrying its numeric code and message.
type ProtocolError struct {
	Code         int
	ErrorMessage string
}

// Error formats the error as the "ERR <code> <message>" wire format.
func (pe ProtocolError) Error() string {
	return fmt.Sprintf("ERR %d %s", pe.Code, pe.ErrorMessage)
}

// Name/login validation errors.
var NameInUseErr = ProtocolError{Code: 201, ErrorMessage: "NAME_IN_USE"}
var NameTooLongErr = ProtocolError{Code: 202, ErrorMessage: "NAME_TOO_LONG"}
var NameTooShortErr = ProtocolError{Code: 202, ErrorMessage: "NAME_TOO_SHORT"}
var InvCharInNameErr = ProtocolError{Code: 204, ErrorMessage: "INVALID_CHAR_IN_NAME"}

// Movement errors.
var NoExitErr = ProtocolError{Code: 301, ErrorMessage: "NO_EXIT"}

// Group errors.
var NotInGroupErr = ProtocolError{Code: 401, ErrorMessage: "NOT_IN_GROUP"}
var InGroupErr = ProtocolError{Code: 402, ErrorMessage: "ALREADY_IN_GROUP"}
var NotLeaderErr = ProtocolError{Code: 403, ErrorMessage: "NOT_GROUP_LEADER"}

// Not-found errors.
var ItemNotFoundErr = ProtocolError{Code: 404, ErrorMessage: "ITEM_NOT_FOUND"}
var NotInInvErr = ProtocolError{Code: 404, ErrorMessage: "ITEM_NOT_IN_INVENTORY"}
var NPCNotFoundErr = ProtocolError{Code: 404, ErrorMessage: "NPC_NOT_FOUND"}
var PlayerNotFoundErr = ProtocolError{Code: 404, ErrorMessage: "PLAYER_NOT_FOUND"}
var NPCNotHostileErr = ProtocolError{Code: 405, ErrorMessage: "NPC_NOT_HOSTILE"}
var NoQuestAvailErr = ProtocolError{Code: 406, ErrorMessage: "NO_QUEST_AVAILABLE"}

// Command parsing errors.
var InvalidCommandErr = ProtocolError{Code: 666, ErrorMessage: "INVALID_COMMAND"}
var InvalidArgsErr = ProtocolError{Code: 670, ErrorMessage: "INVALID_ARGS"}
var MissingArgsErr = ProtocolError{Code: 670, ErrorMessage: "MISSING_ARGS"}

// Rate-limit and ban errors.
var InputSpamErr = ProtocolError{Code: 750, ErrorMessage: "EXCESSIVE_INPUT_DETECTED: TIMEOUT_APPLIED"}
var SoftbannedErr = ProtocolError{Code: 760, ErrorMessage: "SOFTBANNED_FROM_SERVER"}

// Internal and serialization errors.
var InternalErr = ProtocolError{Code: 825, ErrorMessage: "INTERNAL_ERROR"}
var JSONErr = ProtocolError{Code: 880, ErrorMessage: "JSON_ERROR"}
var YAMLErr = ProtocolError{Code: 878, ErrorMessage: "YAML_ERROR"}

// Connection errors.
var ConnFailedErr = ProtocolError{Code: 900, ErrorMessage: "CONNECTION_FAILED"}
var SendFailedErr = ProtocolError{Code: 901, ErrorMessage: "SEND_FAILED"}
var AlreadyConnErr = ProtocolError{Code: 905, ErrorMessage: "ALREADY_CONNECTED"}
var InboundConnErr = ProtocolError{Code: 911, ErrorMessage: "INBOUND_CONNECTION_FAILURE"}

// 201, 301, 401, 402, the three 404s, 405, 406, 900, and 901 CANNOT BE ALTERED!!!
