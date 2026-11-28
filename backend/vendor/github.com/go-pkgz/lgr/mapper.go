package lgr

// Mapper defines optional functions to change elements of the logged message for each part, based on levels.
// Only some mapFunc can be defined, by default does nothing. Can be used to alter the output, for example making some
// part of the output colorful.
type Mapper struct {
	MessageFunc mapFunc // message mapper on all levels, runs before the level's function
	ErrorFunc   mapFunc // level and message mapper on ERROR, PANIC and FATAL levels
	WarnFunc    mapFunc // level and message mapper on WARN level
	InfoFunc    mapFunc // level and message mapper on INFO level
	DebugFunc   mapFunc // level and message mapper on DEBUG and TRACE levels

	CallerFunc mapFunc // caller mapper, all levels
	TimeFunc   mapFunc // time mapper, all levels
}

type mapFunc func(string) string

// nopMapper is a default, doing nothing
var nopMapper = Mapper{
	MessageFunc: func(s string) string { return s },
	ErrorFunc:   func(s string) string { return s },
	WarnFunc:    func(s string) string { return s },
	InfoFunc:    func(s string) string { return s },
	DebugFunc:   func(s string) string { return s },
	CallerFunc:  func(s string) string { return s },
	TimeFunc:    func(s string) string { return s },
}
