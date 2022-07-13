package lexers

import (
	"strings"

	. "github.com/alecthomas/chroma/v2" // nolint
)

// V lexer.
var V = Register(MustNewLexer(
	&Config{
		Name:      "V",
		Aliases:   []string{"v", "vlang"},
		Filenames: []string{"*.v", "*.vv", "v.mod"},
		MimeTypes: []string{"text/x-v"},
		EnsureNL:  true,
	},
	vRules,
).SetAnalyser(func(text string) float32 {
	if strings.Contains(text, "import ") && strings.Contains(text, "module ") {
		return 0.2
	}
	if strings.Contains(text, "module ") {
		return 0.1
	}
	return 0.0
}))

const (
	namePattern             = `[^\W\d]\w*`
	typeNamePattern         = `[A-Z]\w*`
	multiLineCommentPattern = `/\*(?:.|\n)*?\*/`
)

func vRules() Rules {
	return Rules{
		"root": {
			{`\n`, Text, nil},
			{`\s+`, Text, nil},
			{`\\\n`, Text, nil},
			{`(?<=module\s+\w[^\n]*\s+)(//[^\n]+\n)+(?=\n)`, StringDoc, nil},
			{`(// *)(\w+)([^\n]+\n)(?=(?://[^\n]*\n)* *(?:pub +)?(?:fn|struct|union|type|interface|enum|const) +\2\b)`, ByGroups(StringDoc, GenericEmph, StringDoc), Push(`string-doc`)},
			{`//[^\n]*\n`, CommentSingle, nil},
			{`/\*(?:(?:` + multiLineCommentPattern + `)*|.|\n)*\*/`, CommentMultiline, nil},
			{`\b(import|module)\b`, KeywordNamespace, nil},
			{`\b(fn|struct|union|map|chan|type|interface|enum|const|mut|shared|pub|__global)\b`, KeywordDeclaration, nil},
			{`\?`, KeywordDeclaration, nil},
			{`(?<=\)\s*)!`, KeywordDeclaration, nil},
			{`[ \t]*#include[^\n]+`, Using(`c`), nil},
			{`[ \t]*#\w[^\n]*`, CommentPreproc, nil},
			{`(sql)(\s+)(\w+)(\s+)({)([^}]*?)(})`, ByGroups(Keyword, Text, Name, Text, Punctuation, Using(`sql`), Punctuation), nil},
			{`\$(?=\w)`, Operator, nil},
			{`(?<=\$)(?:embed_file|pkgconfig|tmpl|env|compile_error|compile_warn)`, NameBuiltin, nil},
			{`(asm)(\s+)(\w+)(\s*)({)([^}]*?)(})`, ByGroups(Keyword, Text, KeywordType, Text, Punctuation, Using(`nasm`), Punctuation), nil},
			{`\b_(?:un)?likely_(?=\()`, NameFunctionMagic, nil},
			{`(?<=\$if.+?(?:&&|\|\|)?)(` + Words(``, ``, `windows`, `linux`, `macos`, `mac`, `darwin`, `ios`, `android`, `mach`, `dragonfly`, `gnu`, `hpux`, `haiku`, `qnx`, `solaris`, `gcc`, `tinyc`, `clang`, `mingw`, `msvc`, `cplusplus`, `amd64`, `arm64`, `x64`, `x32`, `little_endian`, `big_endian`, `debug`, `prod`, `test`, `js`, `glibc`, `prealloc`, `no_bounds_checking`, `freestanding`, `no_segfault_handler`, `no_backtrace`, `no_main`) + `)+`, NameBuiltin, nil},
			{`@` + Words(``, `\b`, `FN`, `METHOD`, `MOD`, `STRUCT`, `FILE`, `LINE`, `COLUMN`, `VEXE`, `VEXEROOT`, `VHASH`, `VMOD_FILE`, `VMODROOT`), NameVariableMagic, nil},
			{Words(`\b(?<!@)`, `\b`, `break`, `select`, `match`, `defer`, `go`, `goto`, `else`, `if`, `continue`, `for`, `return`, `assert`, `or`, `as`, `atomic`, `isreftype`, `is`, `in`, `lock`, `rlock`, `sizeof`, `typeof`, `unsafe`, `volatile`, `static`, `__offsetof`), Keyword, nil},
			{`\b(?<!@)(none|true|false|si_s_code|si_g32_code|si_g64_code)\b`, KeywordConstant, nil},
			{Words(`\b(?<!@)`, `(?=\()`, `u8`, `u16`, `u32`, `u64`, `u128`, `int`, `i8`, `i16`, `i64`, `i128`, `f32`, `f64`, `rune`, `string`, `bool`, `usize`, `isize`, `any`, `error`, `print`, `println`, `dump`, `panic`, `eprint`, `eprintln`, `copy`, `close`, `len`, `map`, `filter`, `cap`, `delete`, `delete_many`, `delete_last`, `c_error_number_str`, `compare_strings`, `cstring_to_vstring`, `error_with_code`, `exit`, `f32_abs`, `f32_max`, `f32_min`, `f64_max`, `flush_stderr`, `flush_stdout`, `free`, `gc_check_leaks`, `get_str_intp_u32_format`, `get_str_intp_u64_format`, `isnil`, `malloc`, `malloc_noscan`, `memdup`, `memdup_noscan`, `panic_error_number`, `panic_lasterr`, `panic_optional_not_set`, `panic_result_not_set`, `print_backtrace`, `proc_pidpath`, `ptr_str`, `realloc_data`, `str_intp`, `str_intp_g32`, `str_intp_g64`, `str_intp_rune`, `str_intp_sq`, `str_intp_sub`, `string_from_wide`, `string_from_wide2`, `tos`, `tos2`, `tos3`, `tos4`, `tos5`, `tos_clone`, `utf32_decode_to_buffer`, `utf32_to_str`, `utf32_to_str_no_malloc`, `utf8_char_len`, `utf8_getchar`, `utf8_str_visible_length`, `v_realloc`, `vcalloc`, `vcalloc_noscan`, `vmemcmp`, `vmemcpy`, `vmemmove`, `vmemset`, `vstrlen`, `vstrlen_char`, `winapi_lasterr_str`, `reduce`, `string`, `join`, `free`, `join_lines`, `sort_by_len`, `sort_ignore_case`, `str`, `byterune`, `bytestr`, `clone`, `hex`, `utf8_to_utf32`, `vbytes`, `vstring`, `vstring_literal`, `vstring_literal_with_len`, `vstring_with_len`, `try_pop`, `try_push`, `strg`, `strsci`, `strlong`, `eq_epsilon`, `hex_full`, `hex2`, `msg`, `code`, `repeat`, `bytes`, `length_in_bytes`, `ascii_str`, `is_alnum`, `is_bin_digit`, `is_capital`, `is_digit`, `is_hex_digit`, `is_letter`, `is_oct_digit`, `is_space`, `str_escaped`, `repeat_to_depth`, `insert`, `prepend`, `trim`, `drop`, `first`, `last`, `pop`, `clone_to_depth`, `push_many`, `reverse_in_place`, `reverse`, `any`, `all`, `sort`, `sort_with_compare`, `contains`, `index`, `grow_cap`, `grow_len`, `pointers`, `move`, `keys`, `after`, `after_char`, `all_after`, `all_after_last`, `all_before`, `all_before_last`, `before`, `capitalize`, `compare`, `contains_any`, `contains_any_substr`, `count`, `ends_with`, `fields`, `find_between`, `hash`, `index_after`, `index_any`, `index_u8`, `is_lower`, `is_title`, `is_upper`, `last_index`, `last_index_u8`, `len_utf8`, `limit`, `match_glob`, `parse_int`, `parse_uint`, `replace`, `replace_each`, `replace_once`, `runes`, `split`, `split_any`, `split_into_lines`, `split_nth`, `starts_with`, `starts_with_capital`, `strip_margin`, `strip_margin_custom`, `substr`, `substr_ni`, `substr_with_check`, `title`, `to_lower`, `to_upper`, `to_wide`, `trim_left`, `trim_right`, `trim_space`, `trim_string_left`, `trim_string_right`, `utf32_code`), NameBuiltin, nil},
			{Words(`\b(?<!@)`, `\b`, `ArrayFlags`, `AttributeKind`, `ChanState`, `StrIntpType`, `array`, `Error`, `FieldData`, `FunctionData`, `map`, `MethodArgs`, `SortedMap`, `string`, `StrIntpCgenData`, `StrIntpData`, `StrIntpMem`, `StructAttribute`, `VAssertMetaInfo`), NameBuiltin, nil},
			{Words(`\b(?<!@)`, `\b`, `u8`, `u16`, `u32`, `u64`, `u128`, `int`, `i8`, `i16`, `i64`, `i128`, `f32`, `f64`, `rune`, `string`, `bool`, `usize`, `isize`, `any`, `error`, `voidptr`), KeywordType, nil},
			{`\bit\b`, NameVariableMagic, nil},
			{`(?<!fn\s+)(?<=\w+\s+|^)\[(?=:if +)?(?=\w+)`, Punctuation, Push(`attribute`)},
			{`(<<=|>>=|>>>=|>>>|<<|>>|<=|>=|\^=|\+=|-=|\*=|/=|%=|&=|\|=|&&|\|\||<-|\+\+|--|==|!=|:=|\.\.\.|\.\.|[+\-*/%&|^~=#@!])`, Operator, nil},
			{`[\d_]+(\.\d+e[+\-]?\d+|\.\d+|e[+\-]?\d+)`, LiteralNumberFloat, nil},
			{`\.\d+(e[+\-]?\d+)?`, LiteralNumberFloat, nil},
			{`0o[0-7_]+`, LiteralNumberOct, nil},
			{`0x[0-9a-fA-F_]+`, LiteralNumberHex, nil},
			{`0b[01_]+`, LiteralNumberBin, nil},
			{`(0|[1-9][0-9_]*)`, LiteralNumberInteger, nil},
			{"`", StringChar, Push(`char`)},
			Include(`strings`),
			{`@?` + typeNamePattern, NameClass, nil},
			{`(?<=` + namePattern + `)(<)(` + typeNamePattern + `)(>)`, ByGroups(Punctuation, NameClass, Punctuation), nil},
			{`@?` + namePattern + `(?=\()`, NameFunction, nil},
			{`(?<=fn\s+)@?` + namePattern + `(?=\s*\()`, NameFunction, nil},
			{`(?<=(?:continue|break|goto)\s+)\w+`, NameLabel, nil},
			{`\b` + namePattern + `(?=:(?:$|\s+for))`, NameLabel, nil},
			{`[<>()\[\]{}.,;:]`, Punctuation, nil},
			{`@?` + namePattern, NameVariable, nil},
		},
		"strings": {
			{`(c)?(")`, ByGroups(StringAffix, StringDouble), Push(`string-double`)},
			{`(c)?(')`, ByGroups(StringAffix, StringSingle), Push(`string-single`)},
			{`(r)("[^"]+")`, ByGroups(StringAffix, String), nil},
			{`(r)('[^']+')`, ByGroups(StringAffix, String), nil},
		},
		"string-double": {
			{`"`, StringDouble, Pop(1)},
			Include(`char-escape`),
			{`(\$)((?!\\){)`, ByGroups(Operator, Punctuation), Push(`string-curly-interpolation`)},
			{`\$`, Operator, Push(`string-interpolation`)},
			{`[^"]+?`, StringDouble, nil},
		},
		"string-single": {
			{`'`, StringSingle, Pop(1)},
			Include(`char-escape`),
			{`(\$)((?!\\){)`, ByGroups(Operator, Punctuation), Push(`string-curly-interpolation`)},
			{`\$`, Operator, Push(`string-interpolation`)},
			{`[^']+?`, StringSingle, nil},
		},
		"char": {
			{"`", StringChar, Pop(1)},
			Include(`char-escape`),
			{`[^\\]`, StringChar, nil},
		},
		"char-escape": {
			{"\\\\[`'\"\\\\abfnrtv$]|\\\\x[0-9a-fA-F]{2}|\\\\[0-7]{1,3}|\\\\u[0-9a-fA-F]{4}|\\\\U[0-9a-fA-F]{8}", StringEscape, nil},
		},
		"string-doc": {
			{`(// *)(#+ [^\n]+)(\n)`, ByGroups(StringDoc, GenericHeading, Text), nil},
			{`// *([=_*~-])\1{2,}\n`, StringDelimiter, nil},
			{`//[^\n]*\n`, StringDoc, nil},
			Default(Pop(1)),
		},
		"string-interpolation": {
			{`(\.)?(@)?(?:(` + namePattern + `)(\()([^)]*)(\))|(` + namePattern + `))`, ByGroups(Punctuation, Operator, NameFunction, Punctuation, UsingSelf(`root`), Punctuation, NameVariable), nil},
			Default(Pop(1)),
		},
		"string-curly-interpolation": {
			{`}`, Punctuation, Pop(1)},
			Include(`strings`),
			{`(:)( *?)([ 0'#+-])?(?:(\.)?([0-9]+))?([fFgeEGxXobsd])?`, ByGroups(Punctuation, Text, Operator, Punctuation, Number, StringAffix), nil},
			{`[^}"':]+`, UsingSelf(`root`), nil},
		},
		"attribute": {
			{`\]`, Punctuation, Pop(1)},
			{`'`, Punctuation, Push(`string-single`)},
			{`"`, Punctuation, Push(`string-double`)},
			{`[;:]`, Punctuation, nil},
			{`(?<=\[)if\b`, Keyword, nil},
			{`\s+`, Text, nil},
			{`(?<=: *)\w+`, String, nil},
			{namePattern, NameAttribute, nil},
		},
	}
}

// V shell lexer.
var VSH = Register(MustNewLexer(
	&Config{
		Name:      "V shell",
		Aliases:   []string{"vsh", "vshell"},
		Filenames: []string{"*.vsh"},
		MimeTypes: []string{"text/x-vsh"},
		EnsureNL:  true,
	},
	vshRules,
).SetAnalyser(func(text string) float32 {
	firstLine := strings.Split(text, "\n")[0]
	if strings.Contains(firstLine, "#!/usr/bin/env") && strings.Contains(firstLine, "v run") {
		return 1.0
	}
	if strings.Contains(firstLine, "#!/") && strings.Contains(firstLine, "/v run") {
		return 1.0
	}
	return 0.0
}))

func vshRules() Rules {
	vshRules := vRules()
	vshRoot := []Rule{
		{`^#![^\n]*\n`, CommentHashbang, nil},
		{Words(`\b`, `\b`, `args`, `max_path_len`, `wd_at_startup`, `sys_write`, `sys_open`, `sys_close`, `sys_mkdir`, `sys_creat`, `path_separator`, `path_delimiter`, `s_ifmt`, `s_ifdir`, `s_iflnk`, `s_isuid`, `s_isgid`, `s_isvtx`, `s_irusr`, `s_iwusr`, `s_ixusr`, `s_irgrp`, `s_iwgrp`, `s_ixgrp`, `s_iroth`, `s_iwoth`, `s_ixoth`), NameConstant, nil},
		{Words(`\b`, `\b`, `ProcessState`, `SeekMode`, `Signal`, `Command`, `ExecutableNotFoundError`, `File`, `FileNotOpenedError`, `Process`, `Result`, `SizeOfTypeIs0Error`, `Uname`), NameBuiltin, nil},
		{Words(`\b`, `(?=\()`, `abs_path`, `args_after`, `args_before`, `base`, `cache_dir`, `chdir`, `chmod`, `chown`, `config_dir`, `cp`, `cp_all`, `create`, `debugger_present`, `dir`, `environ`, `executable`, `execute`, `execute_or_exit`, `execute_or_panic`, `execve`, `execvp`, `existing_path`, `exists`, `exists_in_system_path`, `expand_tilde_to_home`, `fd_close`, `fd_read`, `fd_slurp`, `fd_write`, `file_ext`, `file_last_mod_unix`, `file_name`, `file_size`, `fileno`, `find_abs_path_of_executable`, `flush`, `fork`, `get_error_msg`, `get_line`, `get_lines`, `get_lines_joined`, `get_raw_line`, `get_raw_lines_joined`, `get_raw_stdin`, `getegid`, `getenv`, `getenv_opt`, `geteuid`, `getgid`, `getpid`, `getppid`, `getuid`, `getwd`, `glob`, `home_dir`, `hostname`, `inode`, `input`, `input_opt`, `is_abs_path`, `is_atty`, `is_dir`, `is_dir_empty`, `is_executable`, `is_file`, `is_link`, `is_readable`, `is_writable`, `is_writable_folder`, `join_path`, `join_path_single`, `last_error`, `link`, `log`, `loginname`, `ls`, `mkdir`, `mkdir_all`, `mv`, `mv_by_cp`, `new_process`, `norm_path`, `open`, `open_append`, `open_file`, `open_uri`, `posix_get_error_msg`, `posix_set_permission_bit`, `quoted_path`, `read_bytes`, `read_file`, `read_file_array`, `read_lines`, `real_path`, `resource_abs_path`, `rm`, `rmdir`, `rmdir_all`, `setenv`, `sigint_to_signal_name`, `signal_opt`, `stderr`, `stdin`, `stdout`, `symlink`, `system`, `temp_dir`, `truncate`, `uname`, `unsetenv`, `user_os`, `utime`, `vfopen`, `vmodules_dir`, `vmodules_paths`, `wait`, `walk`, `walk_ext`, `walk_with_context`, `write_file`, `write_file_array`, `bitmask`, `close`, `read_line`, `start`, `msg`, `read`, `read_bytes_at`, `read_bytes_into`, `read_bytes_into_newline`, `read_from`, `read_into_ptr`, `read_raw`, `read_raw_at`, `read_struct`, `read_struct_at`, `seek`, `tell`, `write`, `write_raw`, `write_raw_at`, `write_string`, `write_struct`, `write_struct_at`, `write_to`, `writeln`, `is_alive`, `run`, `set_args`, `set_environment`, `set_redirect_stdio`, `signal_continue`, `signal_kill`, `signal_pgkill`, `signal_stop`, `stderr_read`, `stderr_slurp`, `stdin_write`, `stdout_read`, `stdout_slurp`), NameBuiltin, nil},
	}

	vshRules[`root`] = append(vshRoot, vshRules[`root`]...)

	return vshRules
}
