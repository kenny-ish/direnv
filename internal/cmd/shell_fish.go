package cmd

import (
	"fmt"
	"strings"
)

type fish struct{}

// Fish adds support for the fish shell as a host
var Fish Shell = fish{}

const fishHook = `
    function __direnv_export_eval --on-event fish_prompt;
        "{{.SelfPath}}" export fish | source;
        __direnv_update_fish_complete_path;

        if test "$direnv_fish_mode" != "disable_arrow";
            function __direnv_cd_hook --on-variable PWD;
                if test "$direnv_fish_mode" = "eval_after_arrow";
                    set -g __direnv_export_again 0;
                else;
                    "{{.SelfPath}}" export fish | source;
                    __direnv_update_fish_complete_path;
                end;
            end;
        end;
    end;

    function __direnv_export_eval_2 --on-event fish_preexec;
        if set -q __direnv_export_again;
            set -e __direnv_export_again;
            "{{.SelfPath}}" export fish | source;
            __direnv_update_fish_complete_path;
            echo;
        end;

        functions --erase __direnv_cd_hook;
    end;

    function __direnv_update_fish_complete_path;
        # Completion paths that should be present for the current XDG_DATA_DIRS.
        # Trailing slashes are stripped so the entries match the ones fish
        # builds itself at startup (/usr/share/ -> /usr/share/fish/vendor_completions.d).
        set -l wanted;
        for dir in (string split ':' -- $XDG_DATA_DIRS);
            set -l completions_dir (string replace -r -- '/+$' '' $dir)"/fish/vendor_completions.d";
            if test -d "$completions_dir";
                set -a wanted $completions_dir;
            end;
        end;

        # Only touch fish_complete_path when something actually changed:
        # fish drops already loaded completions whenever it is modified.
        for p in $__direnv_fish_complete_paths;
            if not contains -- $p $wanted;
                set -l idx (contains -i -- $p $fish_complete_path);
                and set -e fish_complete_path[$idx];
            end;
        end;

        set -l added;
        for completions_dir in $wanted;
            if contains -- $completions_dir $__direnv_fish_complete_paths;
                set -a added $completions_dir;
            else if not contains -- $completions_dir $fish_complete_path;
                set -ga fish_complete_path $completions_dir;
                set -a added $completions_dir;
            end;
        end;
        set -g __direnv_fish_complete_paths $added;
    end;
`

func (sh fish) Hook() (string, error) {
	return fishHook, nil
}

func (sh fish) Export(e ShellExport) (string, error) {
	var out strings.Builder
	for key, value := range e {
		if value == nil {
			out.WriteString(sh.unset(key))
		} else {
			out.WriteString(sh.export(key, *value))
		}
	}
	return out.String(), nil
}

func (sh fish) Dump(env Env) (string, error) {
	var out strings.Builder
	for key, value := range env {
		out.WriteString(sh.export(key, value))
	}
	return out.String(), nil
}

func (sh fish) export(key, value string) string {
	if key == "PATH" {
		var command strings.Builder
		command.WriteString("set -x -g PATH")
		for path := range strings.SplitSeq(value, ":") {
			command.WriteString(" " + sh.escape(path))
		}
		return command.String() + ";"
	}
	return "set -x -g " + sh.escape(key) + " " + sh.escape(value) + ";"
}

func (sh fish) unset(key string) string {
	return "set -e -g " + sh.escape(key) + ";"
}

func (sh fish) escape(str string) string {
	in := []byte(str)
	var out strings.Builder
	out.Grow(len(in) + 2)
	out.WriteByte(SINGLE_QUOTE)
	i := 0
	l := len(in)

	hex := func(char byte) {
		fmt.Fprintf(&out, "'\\X%02x'", char)
	}

	backslash := func(char byte) {
		out.WriteByte(BACKSLASH)
		out.WriteByte(char)
	}

	escaped := func(str string) {
		out.WriteByte(SINGLE_QUOTE)
		out.WriteString(str)
		out.WriteByte(SINGLE_QUOTE)
	}

	literal := func(char byte) {
		out.WriteByte(char)
	}

	for i < l {
		char := in[i]
		switch {
		case char == TAB:
			escaped(`\t`)
		case char == LF:
			escaped(`\n`)
		case char == CR:
			escaped(`\r`)
		case char <= US:
			hex(char)
		case char == SINGLE_QUOTE:
			backslash(char)
		case char == BACKSLASH:
			backslash(char)
		case char <= TILDE:
			literal(char)
		case char == DEL:
			hex(char)
		default:
			hex(char)
		}
		i++
	}

	out.WriteByte(SINGLE_QUOTE)

	return out.String()
}
