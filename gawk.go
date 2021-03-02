// Package gawk is terminal string styling for go done right, with full and painless Windows 10 support.
//
// Gawk is a library heavily inspired by https://github.com/chalk/chalk, the
// popular Node.js terminal color library, and using golang ports of supports-color
// (https://github.com/jwalton/go-supportscolor) and ansi-styles
// (https://github.com/jwalton/gawk/pkg/ansistyles).
//
// A very simple usage example would be:
//
//     fmt.Println(gawk.Blue("This line is blue"))
//
// Note that this works on all platforms - there's no need to write to a special
// stream or use a special print function to get color on Windows 10.
//
// Some examples:
//
//     // Combine styled and normal strings
//     fmt.Println(gawk.Blue("Hello") + " World" + gawk.Red("!"))
//
//     // Compose multiple styles using the chainable API
//     fmt.Println(gawk.WithBlue().WithBgRed().Bold("Hello world!"))
//
//     // Pass in multiple arguments
//     fmt.Println(gawk.Blue("Hello", "World!", "Foo", "bar", "biz", "baz"))
//
//     // Nest styles
//     fmt.Println(gawk.Green(
//         "I am a green line " +
//         gawk.WithBlue().WithUnderline().Bold("with a blue substring") +
//         " that becomes green again!"
//     ))
//
//     // Use RGB colors in terminal emulators that support it.
//     fmt.Println(gawk.WithRGB(123, 45, 67).Underline("Underlined reddish color"))
//     fmt.Println(gawk.WihHex("#DEADED").Bold("Bold gray!"))
//
//     // Write to stderr:
//     os.Stderr.WriteString(gawk.Stderr.Red("Ohs noes!\n"))
//
// See the README.md for more details.
//
package gawk

import (
	"strings"

	"github.com/jwalton/go-supportscolor"
)

type stylerData struct {
	open     string
	close    string
	openAll  string
	closeAll string
	parent   *stylerData
}

type configuration struct {
	Level ColorLevel
}

// A Builder is used to define and chain together styles.
//
// Instances of Builder cannot be constructed directly - you can build a new
// instance via the New() function, which will give you an instance you can
// configure without modifying the "default" Builder.
//
type Builder struct {
	bgBlack         *Builder
	bgBlackBright   *Builder
	bgBlue          *Builder
	bgBlueBright    *Builder
	bgCyan          *Builder
	bgCyanBright    *Builder
	bgGray          *Builder
	bgGreen         *Builder
	bgGreenBright   *Builder
	bgGrey          *Builder
	bgMagenta       *Builder
	bgMagentaBright *Builder
	bgRed           *Builder
	bgRedBright     *Builder
	bgWhite         *Builder
	bgWhiteBright   *Builder
	bgYellow        *Builder
	bgYellowBright  *Builder
	black           *Builder
	blackBright     *Builder
	blue            *Builder
	blueBright      *Builder
	cyan            *Builder
	cyanBright      *Builder
	gray            *Builder
	green           *Builder
	greenBright     *Builder
	grey            *Builder
	magenta         *Builder
	magentaBright   *Builder
	red             *Builder
	redBright       *Builder
	white           *Builder
	whiteBright     *Builder
	yellow          *Builder
	yellowBright    *Builder
	bold            *Builder
	dim             *Builder
	hidden          *Builder
	inverse         *Builder
	italic          *Builder
	overline        *Builder
	strikethrough   *Builder
	underline       *Builder
	reset           *Builder
	styler          *stylerData
	config          *configuration
}

// An Option which can be passed to `New()`.
type Option func(*Builder)

// ForceLevel is an option that can be passed to `New` to force the color level
// used.
func ForceLevel(level ColorLevel) Option {
	return func(builder *Builder) {
		builder.config.Level = level
	}
}

// New creates a new instance of Gawk.
func New(options ...Option) *Builder {
	builder := &Builder{styler: nil}

	builder.config = &configuration{
		Level: ColorLevel(supportscolor.Stdout().Level),
	}

	for index := range options {
		options[index](builder)
	}

	return builder
}

// rootBuilder is the default Gawk instance, pre-configured for stdout.
var rootBuilder = New()

// Stderr is an instance of Gawk pre-configured for stderr.  Use this when coloring
// strings you intend to write the stderr.
var Stderr = New(
	ForceLevel(ColorLevel(supportscolor.Stderr().Level)),
)

func createBuilder(builder *Builder, open string, close string) *Builder {
	var parent *stylerData
	if builder.styler != nil {
		parent = builder.styler
	}

	openAll := open
	closeAll := close
	if parent != nil {
		openAll = parent.openAll + open
		closeAll = close + parent.closeAll
	}

	return &Builder{
		config: builder.config,
		styler: &stylerData{
			open:     open,
			close:    close,
			openAll:  openAll,
			closeAll: closeAll,
			parent:   parent,
		},
	}
}

func (builder *Builder) applyStyle(strs ...string) string {
	if len(strs) == 0 {
		return ""
	}

	str := strings.Join(strs, " ")
	if (builder.config != nil && builder.config.Level <= LevelNone) || str == "" {
		return str
	}

	styler := builder.styler

	if styler == nil {
		return str
	}

	openAll := styler.openAll
	closeAll := styler.closeAll

	if strings.Contains(str, "\u001B") {
		for styler != nil {
			// Replace any instances already present with a re-opening code
			// otherwise only the part of the string until said closing code
			// will be colored, and the rest will simply be 'plain'.
			if styler.close == "\u001b[22m" {
				// This is kind of a weird corner case - both "bold" and "dim"
				// close with "22", but these are actually not mutually exclusive
				// styles - you can have something both bold and dim at the same
				// time (iTerm 2, for example, will render it as a dimmer color,
				// with a bold font face).  So when we nest "dim" inside "bold",
				// if we just replace the dim's close with bold's open, we'll
				// end up with something that is dim and bold at the same time.
				// The fix here is to keep the close tag.  This can lead to
				// a big chain of close tags followed immediately by open tags
				// in cases where we do a lot of nesting, and in any other
				// case this is pointless (as a string can't be both red and
				// blue at the same time, for example), so we treat this as a
				// special case.
				str = strings.ReplaceAll(str, styler.close, styler.close+styler.open)
			} else {
				str = strings.ReplaceAll(str, styler.close, styler.open)
			}

			styler = styler.parent
		}
	}

	// We can move both next actions out of loop, because remaining actions in loop won't have
	// any/visible effect on parts we add here. Close the styling before a linebreak and reopen
	// after next line to fix a bleed issue on macOS: https://github.com/chalk/chalk/pull/92
	if strings.Contains(str, "\n") {
		str = stringEncaseCRLF(str, closeAll, openAll)
	}

	// Concat using "+" instead of fmt.Sprintf, because it's about four times faster.
	return openAll + str + closeAll
}

// SetLevel is used to override the auto-detected color level.
func SetLevel(level ColorLevel) {
	rootBuilder.SetLevel(level)
}

// GetLevel returns the currently configured color level.
func GetLevel() ColorLevel {
	return rootBuilder.GetLevel()
}

// SetLevel is used to override the auto-detected color level for a builder.  Calling
// this at any level of the builder will affect the entire instance of the builder.
func (builder *Builder) SetLevel(level ColorLevel) {
	builder.config.Level = level
}

// GetLevel returns the currently configured level for this builder.
func (builder *Builder) GetLevel() ColorLevel {
	return builder.config.Level
}
