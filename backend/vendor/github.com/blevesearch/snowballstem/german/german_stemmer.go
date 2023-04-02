//! This file was generated automatically by the Snowball to Go compiler
//! http://snowballstem.org/

package german

import (
	snowballRuntime "github.com/blevesearch/snowballstem"
)

var A_0 = []*snowballRuntime.Among{
	{Str: "", A: -1, B: 6, F: nil},
	{Str: "U", A: 0, B: 2, F: nil},
	{Str: "Y", A: 0, B: 1, F: nil},
	{Str: "\u00E4", A: 0, B: 3, F: nil},
	{Str: "\u00F6", A: 0, B: 4, F: nil},
	{Str: "\u00FC", A: 0, B: 5, F: nil},
}

var A_1 = []*snowballRuntime.Among{
	{Str: "e", A: -1, B: 2, F: nil},
	{Str: "em", A: -1, B: 1, F: nil},
	{Str: "en", A: -1, B: 2, F: nil},
	{Str: "ern", A: -1, B: 1, F: nil},
	{Str: "er", A: -1, B: 1, F: nil},
	{Str: "s", A: -1, B: 3, F: nil},
	{Str: "es", A: 5, B: 2, F: nil},
}

var A_2 = []*snowballRuntime.Among{
	{Str: "en", A: -1, B: 1, F: nil},
	{Str: "er", A: -1, B: 1, F: nil},
	{Str: "st", A: -1, B: 2, F: nil},
	{Str: "est", A: 2, B: 1, F: nil},
}

var A_3 = []*snowballRuntime.Among{
	{Str: "ig", A: -1, B: 1, F: nil},
	{Str: "lich", A: -1, B: 1, F: nil},
}

var A_4 = []*snowballRuntime.Among{
	{Str: "end", A: -1, B: 1, F: nil},
	{Str: "ig", A: -1, B: 2, F: nil},
	{Str: "ung", A: -1, B: 1, F: nil},
	{Str: "lich", A: -1, B: 3, F: nil},
	{Str: "isch", A: -1, B: 2, F: nil},
	{Str: "ik", A: -1, B: 2, F: nil},
	{Str: "heit", A: -1, B: 3, F: nil},
	{Str: "keit", A: -1, B: 4, F: nil},
}

var G_v = []byte{17, 65, 16, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 8, 0, 32, 8}

var G_s_ending = []byte{117, 30, 5}

var G_st_ending = []byte{117, 30, 4}

type Context struct {
	i_x  int
	i_p2 int
	i_p1 int
}

func r_prelude(env *snowballRuntime.Env, ctx interface{}) bool {
	context := ctx.(*Context)
	_ = context
	// (, line 33
	// test, line 35
	var v_1 = env.Cursor
	// repeat, line 35
replab0:
	for {
		var v_2 = env.Cursor
	lab1:
		for range [2]struct{}{} {
			// (, line 35
			// or, line 38
		lab2:
			for {
				var v_3 = env.Cursor
			lab3:
				for {
					// (, line 36
					// [, line 37
					env.Bra = env.Cursor
					// literal, line 37
					if !env.EqS("\u00DF") {
						break lab3
					}
					// ], line 37
					env.Ket = env.Cursor
					// <-, line 37
					if !env.SliceFrom("ss") {
						return false
					}
					break lab2
				}
				env.Cursor = v_3
				// next, line 38
				if env.Cursor >= env.Limit {
					break lab1
				}
				env.NextChar()
				break lab2
			}
			continue replab0
		}
		env.Cursor = v_2
		break replab0
	}
	env.Cursor = v_1
	// repeat, line 41
replab4:
	for {
		var v_4 = env.Cursor
	lab5:
		for range [2]struct{}{} {
			// goto, line 41
		golab6:
			for {
				var v_5 = env.Cursor
			lab7:
				for {
					// (, line 41
					if !env.InGrouping(G_v, 97, 252) {
						break lab7
					}
					// [, line 42
					env.Bra = env.Cursor
					// or, line 42
				lab8:
					for {
						var v_6 = env.Cursor
					lab9:
						for {
							// (, line 42
							// literal, line 42
							if !env.EqS("u") {
								break lab9
							}
							// ], line 42
							env.Ket = env.Cursor
							if !env.InGrouping(G_v, 97, 252) {
								break lab9
							}
							// <-, line 42
							if !env.SliceFrom("U") {
								return false
							}
							break lab8
						}
						env.Cursor = v_6
						// (, line 43
						// literal, line 43
						if !env.EqS("y") {
							break lab7
						}
						// ], line 43
						env.Ket = env.Cursor
						if !env.InGrouping(G_v, 97, 252) {
							break lab7
						}
						// <-, line 43
						if !env.SliceFrom("Y") {
							return false
						}
						break lab8
					}
					env.Cursor = v_5
					break golab6
				}
				env.Cursor = v_5
				if env.Cursor >= env.Limit {
					break lab5
				}
				env.NextChar()
			}
			continue replab4
		}
		env.Cursor = v_4
		break replab4
	}
	return true
}

func r_mark_regions(env *snowballRuntime.Env, ctx interface{}) bool {
	context := ctx.(*Context)
	_ = context
	// (, line 47
	context.i_p1 = env.Limit
	context.i_p2 = env.Limit
	// test, line 52
	var v_1 = env.Cursor
	// (, line 52
	{
		// hop, line 52
		var c = env.ByteIndexForHop((3))
		if int32(0) > c || c > int32(env.Limit) {
			return false
		}
		env.Cursor = int(c)
	}
	// setmark x, line 52
	context.i_x = env.Cursor
	env.Cursor = v_1
	// gopast, line 54
golab0:
	for {
	lab1:
		for {
			if !env.InGrouping(G_v, 97, 252) {
				break lab1
			}
			break golab0
		}
		if env.Cursor >= env.Limit {
			return false
		}
		env.NextChar()
	}
	// gopast, line 54
golab2:
	for {
	lab3:
		for {
			if !env.OutGrouping(G_v, 97, 252) {
				break lab3
			}
			break golab2
		}
		if env.Cursor >= env.Limit {
			return false
		}
		env.NextChar()
	}
	// setmark p1, line 54
	context.i_p1 = env.Cursor
	// try, line 55
lab4:
	for {
		// (, line 55
		if !(context.i_p1 < context.i_x) {
			break lab4
		}
		context.i_p1 = context.i_x
		break lab4
	}
	// gopast, line 56
golab5:
	for {
	lab6:
		for {
			if !env.InGrouping(G_v, 97, 252) {
				break lab6
			}
			break golab5
		}
		if env.Cursor >= env.Limit {
			return false
		}
		env.NextChar()
	}
	// gopast, line 56
golab7:
	for {
	lab8:
		for {
			if !env.OutGrouping(G_v, 97, 252) {
				break lab8
			}
			break golab7
		}
		if env.Cursor >= env.Limit {
			return false
		}
		env.NextChar()
	}
	// setmark p2, line 56
	context.i_p2 = env.Cursor
	return true
}

func r_postlude(env *snowballRuntime.Env, ctx interface{}) bool {
	context := ctx.(*Context)
	_ = context
	var among_var int32
	// repeat, line 60
replab0:
	for {
		var v_1 = env.Cursor
	lab1:
		for range [2]struct{}{} {
			// (, line 60
			// [, line 62
			env.Bra = env.Cursor
			// substring, line 62
			among_var = env.FindAmong(A_0, context)
			if among_var == 0 {
				break lab1
			}
			// ], line 62
			env.Ket = env.Cursor
			if among_var == 0 {
				break lab1
			} else if among_var == 1 {
				// (, line 63
				// <-, line 63
				if !env.SliceFrom("y") {
					return false
				}
			} else if among_var == 2 {
				// (, line 64
				// <-, line 64
				if !env.SliceFrom("u") {
					return false
				}
			} else if among_var == 3 {
				// (, line 65
				// <-, line 65
				if !env.SliceFrom("a") {
					return false
				}
			} else if among_var == 4 {
				// (, line 66
				// <-, line 66
				if !env.SliceFrom("o") {
					return false
				}
			} else if among_var == 5 {
				// (, line 67
				// <-, line 67
				if !env.SliceFrom("u") {
					return false
				}
			} else if among_var == 6 {
				// (, line 68
				// next, line 68
				if env.Cursor >= env.Limit {
					break lab1
				}
				env.NextChar()
			}
			continue replab0
		}
		env.Cursor = v_1
		break replab0
	}
	return true
}

func r_R1(env *snowballRuntime.Env, ctx interface{}) bool {
	context := ctx.(*Context)
	_ = context
	if !(context.i_p1 <= env.Cursor) {
		return false
	}
	return true
}

func r_R2(env *snowballRuntime.Env, ctx interface{}) bool {
	context := ctx.(*Context)
	_ = context
	if !(context.i_p2 <= env.Cursor) {
		return false
	}
	return true
}

func r_standard_suffix(env *snowballRuntime.Env, ctx interface{}) bool {
	context := ctx.(*Context)
	_ = context
	var among_var int32
	// (, line 78
	// do, line 79
	var v_1 = env.Limit - env.Cursor
lab0:
	for {
		// (, line 79
		// [, line 80
		env.Ket = env.Cursor
		// substring, line 80
		among_var = env.FindAmongB(A_1, context)
		if among_var == 0 {
			break lab0
		}
		// ], line 80
		env.Bra = env.Cursor
		// call R1, line 80
		if !r_R1(env, context) {
			break lab0
		}
		if among_var == 0 {
			break lab0
		} else if among_var == 1 {
			// (, line 82
			// delete, line 82
			if !env.SliceDel() {
				return false
			}
		} else if among_var == 2 {
			// (, line 85
			// delete, line 85
			if !env.SliceDel() {
				return false
			}
			// try, line 86
			var v_2 = env.Limit - env.Cursor
		lab1:
			for {
				// (, line 86
				// [, line 86
				env.Ket = env.Cursor
				// literal, line 86
				if !env.EqSB("s") {
					env.Cursor = env.Limit - v_2
					break lab1
				}
				// ], line 86
				env.Bra = env.Cursor
				// literal, line 86
				if !env.EqSB("nis") {
					env.Cursor = env.Limit - v_2
					break lab1
				}
				// delete, line 86
				if !env.SliceDel() {
					return false
				}
				break lab1
			}
		} else if among_var == 3 {
			// (, line 89
			if !env.InGroupingB(G_s_ending, 98, 116) {
				break lab0
			}
			// delete, line 89
			if !env.SliceDel() {
				return false
			}
		}
		break lab0
	}
	env.Cursor = env.Limit - v_1
	// do, line 93
	var v_3 = env.Limit - env.Cursor
lab2:
	for {
		// (, line 93
		// [, line 94
		env.Ket = env.Cursor
		// substring, line 94
		among_var = env.FindAmongB(A_2, context)
		if among_var == 0 {
			break lab2
		}
		// ], line 94
		env.Bra = env.Cursor
		// call R1, line 94
		if !r_R1(env, context) {
			break lab2
		}
		if among_var == 0 {
			break lab2
		} else if among_var == 1 {
			// (, line 96
			// delete, line 96
			if !env.SliceDel() {
				return false
			}
		} else if among_var == 2 {
			// (, line 99
			if !env.InGroupingB(G_st_ending, 98, 116) {
				break lab2
			}
			{
				// hop, line 99
				var c = env.ByteIndexForHop(-(3))
				if int32(env.LimitBackward) > c || c > int32(env.Limit) {
					break lab2
				}
				env.Cursor = int(c)
			}
			// delete, line 99
			if !env.SliceDel() {
				return false
			}
		}
		break lab2
	}
	env.Cursor = env.Limit - v_3
	// do, line 103
	var v_4 = env.Limit - env.Cursor
lab3:
	for {
		// (, line 103
		// [, line 104
		env.Ket = env.Cursor
		// substring, line 104
		among_var = env.FindAmongB(A_4, context)
		if among_var == 0 {
			break lab3
		}
		// ], line 104
		env.Bra = env.Cursor
		// call R2, line 104
		if !r_R2(env, context) {
			break lab3
		}
		if among_var == 0 {
			break lab3
		} else if among_var == 1 {
			// (, line 106
			// delete, line 106
			if !env.SliceDel() {
				return false
			}
			// try, line 107
			var v_5 = env.Limit - env.Cursor
		lab4:
			for {
				// (, line 107
				// [, line 107
				env.Ket = env.Cursor
				// literal, line 107
				if !env.EqSB("ig") {
					env.Cursor = env.Limit - v_5
					break lab4
				}
				// ], line 107
				env.Bra = env.Cursor
				// not, line 107
				var v_6 = env.Limit - env.Cursor
			lab5:
				for {
					// literal, line 107
					if !env.EqSB("e") {
						break lab5
					}
					env.Cursor = env.Limit - v_5
					break lab4
				}
				env.Cursor = env.Limit - v_6
				// call R2, line 107
				if !r_R2(env, context) {
					env.Cursor = env.Limit - v_5
					break lab4
				}
				// delete, line 107
				if !env.SliceDel() {
					return false
				}
				break lab4
			}
		} else if among_var == 2 {
			// (, line 110
			// not, line 110
			var v_7 = env.Limit - env.Cursor
		lab6:
			for {
				// literal, line 110
				if !env.EqSB("e") {
					break lab6
				}
				break lab3
			}
			env.Cursor = env.Limit - v_7
			// delete, line 110
			if !env.SliceDel() {
				return false
			}
		} else if among_var == 3 {
			// (, line 113
			// delete, line 113
			if !env.SliceDel() {
				return false
			}
			// try, line 114
			var v_8 = env.Limit - env.Cursor
		lab7:
			for {
				// (, line 114
				// [, line 115
				env.Ket = env.Cursor
				// or, line 115
			lab8:
				for {
					var v_9 = env.Limit - env.Cursor
				lab9:
					for {
						// literal, line 115
						if !env.EqSB("er") {
							break lab9
						}
						break lab8
					}
					env.Cursor = env.Limit - v_9
					// literal, line 115
					if !env.EqSB("en") {
						env.Cursor = env.Limit - v_8
						break lab7
					}
					break lab8
				}
				// ], line 115
				env.Bra = env.Cursor
				// call R1, line 115
				if !r_R1(env, context) {
					env.Cursor = env.Limit - v_8
					break lab7
				}
				// delete, line 115
				if !env.SliceDel() {
					return false
				}
				break lab7
			}
		} else if among_var == 4 {
			// (, line 119
			// delete, line 119
			if !env.SliceDel() {
				return false
			}
			// try, line 120
			var v_10 = env.Limit - env.Cursor
		lab10:
			for {
				// (, line 120
				// [, line 121
				env.Ket = env.Cursor
				// substring, line 121
				among_var = env.FindAmongB(A_3, context)
				if among_var == 0 {
					env.Cursor = env.Limit - v_10
					break lab10
				}
				// ], line 121
				env.Bra = env.Cursor
				// call R2, line 121
				if !r_R2(env, context) {
					env.Cursor = env.Limit - v_10
					break lab10
				}
				if among_var == 0 {
					env.Cursor = env.Limit - v_10
					break lab10
				} else if among_var == 1 {
					// (, line 123
					// delete, line 123
					if !env.SliceDel() {
						return false
					}
				}
				break lab10
			}
		}
		break lab3
	}
	env.Cursor = env.Limit - v_4
	return true
}

func Stem(env *snowballRuntime.Env) bool {
	var context = &Context{
		i_x:  0,
		i_p2: 0,
		i_p1: 0,
	}
	_ = context
	// (, line 133
	// do, line 134
	var v_1 = env.Cursor
lab0:
	for {
		// call prelude, line 134
		if !r_prelude(env, context) {
			break lab0
		}
		break lab0
	}
	env.Cursor = v_1
	// do, line 135
	var v_2 = env.Cursor
lab1:
	for {
		// call mark_regions, line 135
		if !r_mark_regions(env, context) {
			break lab1
		}
		break lab1
	}
	env.Cursor = v_2
	// backwards, line 136
	env.LimitBackward = env.Cursor
	env.Cursor = env.Limit
	// do, line 137
	var v_3 = env.Limit - env.Cursor
lab2:
	for {
		// call standard_suffix, line 137
		if !r_standard_suffix(env, context) {
			break lab2
		}
		break lab2
	}
	env.Cursor = env.Limit - v_3
	env.Cursor = env.LimitBackward
	// do, line 138
	var v_4 = env.Cursor
lab3:
	for {
		// call postlude, line 138
		if !r_postlude(env, context) {
			break lab3
		}
		break lab3
	}
	env.Cursor = v_4
	return true
}
