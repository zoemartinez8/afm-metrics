package afm

// runeToGlyphName maps a Unicode code point to the PostScript glyph name
// Adobe's StandardEncoding (and, for the accented Latin letters, the wider
// convention every base-14 Type 1 font follows for its unencoded glyphs)
// assigns it. This is what makes GlyphName useful for anything past plain
// ASCII: StandardEncoding's own character codes only line up with Unicode
// code points for codes 32-126. Past that, the two diverge on purpose - for
// example U+2014 EM DASH is glyph "emdash" at StandardEncoding code 208,
// and U+00E9 (e-acute) has no code of its own at all, only the name
// "eacute" a font can carry as an unencoded glyph.
//
// Spacing diacritics (grave, acute, circumflex, tilde, dieresis, and so on)
// are deliberately left out: the Adobe Glyph List maps most of them to the
// same code point as an ASCII punctuation mark (e.g. "grave" and
// "quoteleft" both claim U+0060), and resolving that ambiguity in favor of
// the accent glyph would silently break the far more common backtick case.
var runeToGlyphName = map[rune]string{
	' ': "space", '!': "exclam", '"': "quotedbl", '#': "numbersign",
	'$': "dollar", '%': "percent", '&': "ampersand", '\'': "quoteright",
	'(': "parenleft", ')': "parenright", '*': "asterisk", '+': "plus",
	',': "comma", '-': "hyphen", '.': "period", '/': "slash",
	'0': "zero", '1': "one", '2': "two", '3': "three", '4': "four",
	'5': "five", '6': "six", '7': "seven", '8': "eight", '9': "nine",
	':': "colon", ';': "semicolon", '<': "less", '=': "equal",
	'>': "greater", '?': "question", '@': "at",
	'A': "A", 'B': "B", 'C': "C", 'D': "D", 'E': "E", 'F': "F", 'G': "G",
	'H': "H", 'I': "I", 'J': "J", 'K': "K", 'L': "L", 'M': "M", 'N': "N",
	'O': "O", 'P': "P", 'Q': "Q", 'R': "R", 'S': "S", 'T': "T", 'U': "U",
	'V': "V", 'W': "W", 'X': "X", 'Y': "Y", 'Z': "Z",
	'[': "bracketleft", '\\': "backslash", ']': "bracketright",
	'^': "asciicircum", '_': "underscore", '`': "quoteleft",
	'a': "a", 'b': "b", 'c': "c", 'd': "d", 'e': "e", 'f': "f", 'g': "g",
	'h': "h", 'i': "i", 'j': "j", 'k': "k", 'l': "l", 'm': "m", 'n': "n",
	'o': "o", 'p': "p", 'q': "q", 'r': "r", 's': "s", 't': "t", 'u': "u",
	'v': "v", 'w': "w", 'x': "x", 'y': "y", 'z': "z",
	'{': "braceleft", '|': "bar", '}': "braceright", '~': "asciitilde",

	// StandardEncoding's high half (octal 241-373 / decimal 161-251):
	// typographic punctuation with no ASCII equivalent.
	'¡': "exclamdown", '¢': "cent", '£': "sterling",
	'⁄': "fraction", '¥': "yen", 'ƒ': "florin",
	'§': "section", '¤': "currency", '“': "quotedblleft",
	'«': "guillemotleft", '‹': "guilsinglleft",
	'›': "guilsinglright", 'ﬁ': "fi", 'ﬂ': "fl",
	'–': "endash", '†': "dagger", '‡': "daggerdbl",
	'·': "periodcentered", '¶': "paragraph", '•': "bullet",
	'‚': "quotesinglbase", '„': "quotedblbase",
	'”': "quotedblright", '»': "guillemotright",
	'…': "ellipsis", '‰': "perthousand", '¿': "questiondown",
	'—': "emdash", 'Æ': "AE", 'ª': "ordfeminine",
	'Ł': "Lslash", 'Ø': "Oslash", 'Œ': "OE",
	'º': "ordmasculine", 'æ': "ae", 'ı': "dotlessi",
	'ł': "lslash", 'ø': "oslash", 'œ': "oe",
	'ß': "germandbls",

	// Accented Latin letters: not in StandardEncoding's own repertoire, but
	// every base-14 font carries them as unencoded (C -1) glyphs under
	// these names, often built as composites via CC/PCC.
	'À': "Agrave", 'Á': "Aacute", 'Â': "Acircumflex",
	'Ã': "Atilde", 'Ä': "Adieresis", 'Å': "Aring",
	'Ç': "Ccedilla", 'È': "Egrave", 'É': "Eacute",
	'Ê': "Ecircumflex", 'Ë': "Edieresis", 'Ì': "Igrave",
	'Í': "Iacute", 'Î': "Icircumflex", 'Ï': "Idieresis",
	'Ñ': "Ntilde", 'Ò': "Ograve", 'Ó': "Oacute",
	'Ô': "Ocircumflex", 'Õ': "Otilde", 'Ö': "Odieresis",
	'Ù': "Ugrave", 'Ú': "Uacute", 'Û': "Ucircumflex",
	'Ü': "Udieresis", 'Ý': "Yacute", 'Þ': "Thorn",
	'à': "agrave", 'á': "aacute", 'â': "acircumflex",
	'ã': "atilde", 'ä': "adieresis", 'å': "aring",
	'ç': "ccedilla", 'è': "egrave", 'é': "eacute",
	'ê': "ecircumflex", 'ë': "edieresis", 'ì': "igrave",
	'í': "iacute", 'î': "icircumflex", 'ï': "idieresis",
	'ñ': "ntilde", 'ò': "ograve", 'ó': "oacute",
	'ô': "ocircumflex", 'õ': "otilde", 'ö': "odieresis",
	'ù': "ugrave", 'ú': "uacute", 'û': "ucircumflex",
	'ü': "udieresis", 'ý': "yacute", 'þ': "thorn",
	'ÿ': "ydieresis",
}

// GlyphName returns the PostScript glyph name r resolves to under Adobe's
// StandardEncoding convention, and whether r has a known mapping. Most of
// Unicode reports false: an AFM file only names the glyphs its font
// actually ships.
func GlyphName(r rune) (string, bool) {
	name, ok := runeToGlyphName[r]
	return name, ok
}
