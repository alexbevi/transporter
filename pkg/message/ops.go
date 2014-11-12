package message

// OpType represents the many different Operations being
// performed against a document (i.e. Insert, Update, etc.)
type OpType int

const (
	Insert OpType = iota
	Update
	Delete
	Command
	Unknown
)

// String returns the constant of the
// string representation of the OpType object.
func (o OpType) String() string {
	switch o {
	case Insert:
		return "insert"
	case Update:
		return "update"
	case Delete:
		return "delete"
	case Command:
		return "command"
	default:
		return "unknown"
	}
}

// OpTypeFromString returns the constant
// representing the passed in string
func OpTypeFromString(s string) OpType {
	switch s[0] {
	case 'i':
		return Insert
	case 'u':
		return Update
	case 'd':
		return Delete
	case 'c':
		return Command
	default:
		return Unknown
	}
}

// CommandType represents the different Commands capable
// of being executed against a database.
type CommandType int

const (
	Flush CommandType = iota
)
