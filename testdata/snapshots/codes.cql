library Differential version '1.0'
codesystem "CS": 'a' version 'b'
codesystem "S": 'a'
code "X": 'a' from "CS"
code "Y": 'a' from "S" display ''
code "Z": 'A' from "S" display 'b'
concept "K": { "X", "Z" } display ''
concept "L": { "Y" }
define D0: (Code { code: 'A', system: null, version: null } = Code { code: '', system: null, version: 'A' })
define D1: ({ (null as Code), Code { code: 'A', system: 'A', display: '', version: 'A' } } = { (null as Code), Code { code: 'b', display: 'a', version: null } })
define D2: ({ Code { code: '', system: 'a', display: null, version: 'a' } } != { Code { code: '', system: 'b', display: null, version: 'a' } })
define D3: ((Code 'A' from "S" display 'a').system is null)
define D4: (Concept { codes: { Code { code: 'a', system: null, display: '', version: 'a' }, Code { code: 'A', system: null, display: 'A', version: 'A' } }, display: 'A' } !~ Concept { codes: { Code { code: '', system: 'b', version: null }, Code { code: 'b', display: null, version: null } }, display: null })
define D5: (Concept { codes: { Code { code: '', system: 'a', display: null, version: '' } } } != Concept { codes: { Code { code: 'A', system: 'a', display: null, version: '' } } })
define D6: (Code { code: null, system: 'b', display: 'b', version: 'A' } !~ Code { code: null, system: 'b', display: 'b', version: 'A' })
define D7: (Code { code: 'b', version: 'a' } ~ Code { code: 'b', version: 'b' })
define D8: (Code { system: '', display: 'b', version: null } !~ Code { system: '', display: 'b', version: 'b' })
define D9: (Concept { codes: { Code { code: 'a', version: null } }, display: 'A' }).display
define D10: (Code { code: 'b', system: 'A', version: '' } = Code { code: 'A', system: 'A', version: '' })
define D11: (Code 'a' from "CS" display 'a').version
define D12: ({ Code { code: 'A', system: null, version: 'a' } } = { Code { code: 'A', system: null, display: 'A', version: 'a' } })
define D13: ({ Code { code: 'b', system: 'a', display: 'A', version: 'b' } } != { Code { code: 'A' } })
define D14: (Code { code: 'A', system: null, display: null, version: 'a' } = Code { code: 'A', system: null, display: null, version: 'a' })
define D15: (Concept { codes: { Code { code: null, system: '', display: null, version: 'a' } } } = Concept { codes: { Code { system: 'b', version: null } } })
define D16: (Code '' from "CS" display 'a').version
define D17: ("Y").code
define D18: (Code { code: 'a', version: 'a' }).display
define D19: ((Concept { codes: { Code { code: 'b', system: 'a', display: null, version: '' } }, display: 'b' }).display is null)
define D20: ({ Code { system: null, display: 'a', version: 'b' } } != { Code { code: null, display: null } })
define D21: (Code { code: 'A', system: null, display: '' } != Code { code: 'a', version: 'a' })
define D22: ("K" != Concept { codes: { Code { code: '', display: 'a', version: 'A' } } })
define D23: (Code { code: 'a', system: '', display: null } !~ Code { code: 'a', system: 'A', version: '' })
define D24: ((Concept { codes: { Code { code: 'b', system: 'b', display: null, version: 'b' } }, display: 'A' }).display is null)
define D25: (Code { code: 'a', display: 'a' }).display
define D26: ((Concept { codes: { Code { code: 'A', system: null, display: 'b', version: 'a' } }, display: '' }).display is null)
define D27: ((Concept { codes: { Code { code: 'a', system: 'A', display: null, version: null } }, display: 'a' }).display is null)
define D28: (Code { code: 'a', system: 'a', version: 'A' } != Code { code: 'a', system: 'a', version: '' })
define D29: (Concept { codes: { Code { code: 'a', version: 'A' }, Code { code: '', system: 'A', display: '', version: 'a' } }, display: 'b' }).display
define D30: (Concept { codes: { Code { code: 'b', display: null } }, display: 'A' }).display
define D31: (Concept { codes: { Code { system: null, display: 'b', version: 'A' }, Code { system: 'a', display: 'b', version: null } }, display: '' } !~ Concept { codes: { Code { code: 'A', system: null, display: 'b', version: 'A' }, Code { code: null, display: 'b', version: 'b' } }, display: '' })
define D32: (Concept { codes: { Code { code: 'A', system: 'a', display: null, version: '' }, Code 'a' from "S" display 'b' }, display: 'A' }).display
define D33: ((Code '' from "CS" display 'A').code is null)
define D34: (Code { code: null, system: 'a', display: 'b', version: '' } !~ Code { code: null, system: 'b', display: 'a', version: 'b' })
define D35: ((Code { code: '', display: 'a', version: 'b' }).code is null)
define D36: (Code { code: 'b', system: 'A', version: 'A' } != Code { code: '', system: 'a', display: 'A', version: 'A' })
define D37: ((Concept { codes: { Code 'a' from "CS" display 'b' }, display: null }).display is null)
define D38: (Concept { codes: { Code { code: null, version: null } }, display: 'a' } != Concept { codes: { Code { system: '', display: '', version: 'a' } }, display: '' })
define D39: (("Z").code is null)
define D40: (Code { code: null, system: null, display: 'a', version: 'a' } != Code { code: 'A', system: 'b', display: 'a', version: '' })
define D41: (Code { code: 'b', system: 'A', display: null, version: 'A' }).code
define D42: ((null as Code) != Code { code: '', system: '' })
define D43: (Code { code: 'b', version: 'a' }).code
define D44: (Code 'A' from "CS" display '' != Code { code: null, system: 'a', version: null })
define D45: (Code { code: '', system: 'A', display: 'a', version: 'b' } != Code { code: 'b', system: 'A', display: 'a', version: 'b' })
define D46: (Concept { codes: { Code { code: 'A', system: 'a', display: 'a', version: 'b' } }, display: null } != Concept { codes: { Code { code: null, system: 'b', display: null, version: 'A' } }, display: null })
define D47: (Code '' from "CS" display 'A').code
define D48: (Code 'A' from "S" display 'b').version
define D49: ((Code { code: 'a', system: 'a', version: null }).system is null)
define D50: (Concept { codes: { Code { code: 'b', display: '', version: '' } }, display: 'b' } = Concept { codes: { Code { code: '', system: '', display: 'A', version: 'a' } }, display: 'b' })
define D51: ("L").display
define D52: (Code 'A' from "CS" display 'b').system
define D53: (Concept { codes: { Code { code: 'b', display: '' }, Code { code: 'A', system: 'b', display: null } }, display: null }).display
define D54: (("X").display is null)
define D55: ({ Code { version: null } } != { Code { version: null } })
define D56: (Concept { codes: { Code { code: 'b', system: 'b', display: '', version: 'b' }, Code { code: '', system: null, display: 'b' } }, display: 'a' } !~ "L")
define D57: (("X").code is null)
define D58: (Code 'a' from "S").code
define D59: ((Code 'A' from "S" display 'A').version is null)
define D60: ((null as Code) = Code { system: null, display: '', version: null })
define D61: (Concept { codes: { Code { code: null, system: 'a' }, Code { code: 'A', system: 'a', display: 'a', version: null } }, display: 'a' } !~ Concept { codes: { Code { code: '', system: null, display: 'A', version: null }, Code { code: 'A', system: 'a', display: null, version: null } }, display: 'a' })
define D62: ("K").display
define D63: (Code { code: null } ~ Code { code: null, display: 'A' })
define D64: (Code { code: '', system: 'a', version: '' } ~ Code { code: '', system: 'a', display: 'A', version: '' })
define D65: ("L" ~ Concept { codes: { Code { system: null, display: 'a', version: 'b' } }, display: 'b' })
define D66: (Code { code: 'a', system: 'a', version: '' } != Code { code: 'a', system: 'a', display: 'b', version: '' })
define D67: ({ (null as Code), Code 'A' from "S" } != { (null as Code), Code { code: 'b', system: null, display: null } })
define D68: (Code { system: 'b', version: '' } ~ Code { code: '', system: 'A', display: null, version: 'A' })
define D69: (Code { code: 'b', system: 'b', display: 'a', version: null }).system
define D70: (Code { code: null, display: '' } = Code { code: '', system: null, display: null, version: 'A' })
define D71: ((null as Code) !~ Code { code: 'a', system: 'b', display: null, version: 'b' })
define D72: (Concept { codes: { Code { system: 'b', display: 'a', version: '' }, Code { code: 'A', system: 'b', display: '', version: 'b' } }, display: 'b' } !~ Concept { codes: { Code { system: null, display: 'a', version: '' }, Code { code: 'A', system: 'b', display: 'b', version: 'b' } }, display: 'b' })
define D73: (Concept { codes: { Code { code: null, system: null, display: null, version: '' } } }).display
define D74: ((Code '' from "CS").code is null)
define D75: (Code { code: 'A', display: 'A', version: 'a' }).system
define D76: (Code { code: 'A', system: '', display: 'a', version: null } != Code { code: 'A', system: null, display: 'a', version: null })
define D77: (Code 'a' from "S" display 'b').system
define D78: (Code { code: 'a', display: '', version: '' } = Code { code: 'a', display: '', version: 'a' })
define D79: (Concept { codes: { Code { code: '', system: 'A', display: 'b', version: null } }, display: 'b' }).display
define D80: ((null as Concept)).display
define D81: (Code { code: 'A', system: 'b', display: 'A', version: 'a' } ~ Code { code: 'A', display: 'A', version: 'a' })
define D82: (Concept { codes: { Code { system: null }, Code { system: '', display: 'b', version: null } }, display: 'A' }).display
define D83: ((Code 'A' from "S").display is null)
define D84: (Code { code: 'a', system: '', display: 'a', version: '' } = Code '' from "S" display 'b')
define D85: (Concept { codes: { Code { code: 'b', display: 'A', version: null } }, display: '' } != Concept { codes: { Code { code: '', display: 'A', version: null } }, display: 'a' })
define D86: ({ (null as Code), Code { code: 'A', system: 'A', display: 'b' } } != { (null as Code), Code { code: 'A', system: 'A', display: 'b' } })
define D87: (Code { code: '', system: 'a', display: 'b', version: null } = Code { system: 'a', display: 'b', version: null })
define D88: (Code { system: null, display: 'A', version: null }).version
define D89: (Code { code: 'a', system: '', display: 'b', version: 'b' } != Code { code: 'a', system: '', display: 'b' })
define D90: (Code { code: 'a', system: 'a', display: '', version: '' } != Code { code: 'a', system: 'a', display: '', version: '' })
define D91: ({ Code { code: 'b', system: null, display: 'A', version: null } } != { Code { code: null, system: null, display: '', version: 'b' } })
define D92: (Concept { codes: { Code { code: 'b', system: 'a', display: 'b', version: 'A' }, Code { code: 'A', display: 'A', version: '' } }, display: null } !~ Concept { codes: { Code { code: 'A', system: 'A', version: 'A' }, Code { code: 'b', display: '', version: 'b' } } })
define D93: ((Concept { codes: { Code { code: 'a', system: 'b', display: '', version: null } }, display: 'A' }).display is null)
define D94: (Code '' from "S" display 'b' != Code { code: 'b', system: 'b', display: null, version: 'a' })
define D95: (Code { code: 'b', system: 'A', display: 'a', version: 'b' } ~ Code { code: 'b', display: 'a', version: 'b' })
define D96: (Concept { codes: { Code { code: '', system: 'b', display: '', version: null }, Code { code: '', system: 'b', version: 'A' } }, display: 'b' } ~ "K")
define D97: (Code { code: null, display: 'A', version: 'a' } !~ Code { code: null, display: 'A' })
define D98: (Concept { codes: { Code { code: 'b', system: 'A', display: 'b' }, Code { code: 'a', system: '', display: 'A' } } }).display
define D99: (((null as Concept)).display is null)
define D100: (Concept { codes: { Code { system: 'A', display: 'a', version: '' }, Code { system: 'A', display: 'b' } }, display: '' } !~ Concept { codes: { Code { system: 'A', display: 'a', version: 'b' }, Code { system: 'A', display: 'b' } }, display: null })
define D101: (Concept { codes: { Code { code: '', system: 'A', display: 'a' } } } != Concept { codes: { Code { system: '', display: 'A', version: 'A' } }, display: '' })
define D102: (Code { code: null, system: 'a', display: 'A', version: '' }).display
define D103: ({ Code { code: 'a', system: '', version: 'A' } } != { Code { code: 'a', system: '', version: null } })
define D104: ({ Code { code: 'A', system: null, version: null } } = { Code { code: 'A', system: '', display: 'a', version: 'b' } })
define D105: ({ (null as Code), Code { code: 'A', system: 'b', display: 'a', version: 'a' } } = { (null as Code), Code { code: 'A', system: 'b', version: 'a' } })
define D106: (Code { code: null, system: 'A', display: '' } ~ Code { code: 'a', version: 'a' })
define D107: (Code { system: '', display: 'A', version: 'A' } !~ Code { code: 'b', system: '', display: 'a', version: '' })
define D108: (Code { system: '', display: 'b', version: '' } = Code { system: 'a', version: '' })
define D109: ((Code 'a' from "S" display 'b').version is null)
define D110: (Code { code: 'a', display: 'b', version: 'a' } !~ Code { code: null, system: 'a', display: 'a', version: 'b' })
define D111: (Code { code: 'b', system: '', display: null } ~ Code { code: 'b', system: null, display: null })
define D112: ({ Code { code: 'A' } } != { Code { code: 'a', system: 'a', display: 'b', version: null } })
define D113: (Code { system: 'A', display: 'a', version: null } ~ Code { system: 'A', display: 'a', version: 'a' })
define D114: (("K").display is null)
define D115: (Code { code: 'b', display: 'A', version: 'b' } != Code { code: 'b', system: 'A', display: 'A', version: 'b' })
define D116: (Code { code: 'A', system: 'A', display: 'a', version: 'a' } = Code { code: 'b', system: 'A', display: 'a', version: 'a' })
define D117: (Concept { codes: { Code { code: '', system: '', display: 'A', version: '' }, Code { code: 'b', system: 'A' } }, display: null }).display
define D118: ((Concept { codes: { Code { code: 'a', system: 'b', display: 'b', version: 'b' } }, display: 'A' }).display is null)
define D119: (Concept { codes: { Code { code: null, system: 'A', display: null, version: '' } }, display: null } = Concept { codes: { Code { code: 'b', display: 'b' } }, display: null })
define D120: (Code { code: null, system: null, display: 'A', version: 'A' } = Code { display: 'a', version: 'b' })
define D121: (Concept { codes: { Code { code: '', system: 'b', version: 'b' } }, display: null } != Concept { codes: { Code { code: 'a', system: null } }, display: 'a' })
define D122: (Concept { codes: { Code { system: 'a', display: 'a', version: '' } }, display: null } != Concept { codes: { Code { code: '', system: 'A', display: '', version: 'b' } }, display: 'b' })
define D123: ((Code '' from "S").system is null)
define D124: (Code '' from "CS").version
define D125: (Concept { codes: { Code { code: 'a', system: '', display: null, version: 'A' } } }).display
define D126: ((Code 'A' from "CS").version is null)
define D127: (Code 'a' from "CS").display
define D128: (Concept { codes: { Code { code: null, system: 'b', display: 'a', version: '' } }, display: '' } != Concept { codes: { Code { code: '', system: '', version: null } }, display: null })
define D129: ((Code { code: 'A', system: 'a', display: '' }).code is null)
define D130: (Code { system: null, display: 'a', version: 'b' } !~ Code { version: 'b' })
define D131: ((Concept { codes: { Code { code: 'A', system: 'a' } } }).display is null)
define D132: (Code { code: 'a', system: 'a', display: 'a', version: 'b' } !~ Code { code: 'a', system: null, display: 'a', version: 'b' })
define D133: ({ (null as Code), Code { code: 'A', system: 'b', display: 'A', version: '' } } = { (null as Code), Code { code: '', system: 'b', display: 'A', version: '' } })
define D134: (Code { display: 'b', version: 'b' } !~ Code { code: 'a', system: 'A', display: null, version: 'a' })
define D135: (Code { code: 'a', system: null, display: '', version: null } = "Z")
define D136: (Code { system: 'a', display: '', version: null } !~ Code { system: null, display: '', version: null })
define D137: (Code '' from "CS" display '').display
define D138: (Code { code: 'A', system: 'a', display: '', version: 'A' } = Code { code: null, system: '', version: null })
define D139: (Code 'a' from "CS" = Code { code: 'A', system: '', display: 'b', version: null })
define D140: ((Concept { codes: { Code { code: 'a', system: '', display: 'b' } } }).display is null)
define D141: (("X").system is null)
define D142: ("Z" = Code { code: 'a', system: 'a', display: 'A', version: null })
define D143: (Code { code: 'a', system: 'b', version: '' } != Code { code: null, system: 'b', version: '' })
define D144: (Concept { codes: { Code { code: 'a', system: 'a', display: 'a', version: null }, Code { code: 'A', system: 'b', display: null } } } !~ Concept { codes: { Code { code: 'b', system: null, display: null, version: 'b' }, Code { code: null, system: null, version: '' } }, display: null })
define D145: (Concept { codes: { Code { code: '', system: null, display: '', version: '' } }, display: null } != Concept { codes: { Code { code: '', display: null, version: 'a' } }, display: 'a' })
define D146: ({ Code { code: 'a', system: 'a', display: null, version: 'A' } } = { Code { code: 'b', system: null, version: '' } })
define D147: (Code { system: null, display: 'A', version: '' } ~ Code { system: '', display: 'A', version: 'b' })
define D148: (("Y").version is null)
define D149: (Code { code: null, version: 'a' } != Code { code: 'a', display: null })
define D150: (Code { code: 'b', system: 'b', version: null } ~ Code { code: 'b', system: 'b', version: null })
define D151: ((Concept { codes: { Code { code: 'b', system: 'A', display: 'a', version: 'A' } }, display: 'a' }).display is null)
define D152: (Concept { codes: { Code { code: 'a', system: null, display: 'b', version: null } } } != Concept { codes: { Code { code: 'b', system: 'b', display: 'A', version: '' } } })
define D153: ({ (null as Code), Code { code: 'a', system: null, version: 'a' } } = { (null as Code), Code { code: '', system: 'b', display: null, version: 'a' } })
define D154: (Concept { codes: { Code { system: 'A', display: 'b', version: 'b' }, Code { code: 'a', system: 'b', display: 'a', version: '' } }, display: 'b' } !~ Concept { codes: { Code { code: 'a', system: 'b', version: '' }, Code { code: 'a', system: null, display: null, version: '' } }, display: 'b' })
define D155: (Concept { codes: { Code { code: 'b', system: 'a', display: '', version: 'A' } }, display: 'b' } ~ Concept { codes: { Code { code: 'b', system: 'a', display: 'b', version: '' }, Code { code: 'a', system: '', display: '', version: '' } }, display: 'A' })
define D156: ((Code '' from "CS").display is null)
define D157: (Code { code: 'a', system: 'a', display: 'a', version: 'A' } ~ Code { code: 'a', display: 'b', version: 'A' })
define D158: ({ Code { code: '', system: '', version: 'b' } } = { Code { code: '', system: '' } })
define D159: (Concept { codes: { Code { code: 'a', system: 'A', display: '', version: 'a' }, Code { code: 'b', system: '', display: 'A', version: '' } }, display: 'b' } ~ Concept { codes: { Code { code: 'a', system: 'A', version: 'a' }, Code 'a' from "S" }, display: 'b' })
define D160: ((Concept { codes: { Code { code: 'b', system: '', display: '', version: 'A' }, Code { code: 'A', system: null, display: null, version: null } } }).display is null)
define D161: (Concept { codes: { Code { code: '', display: '', version: 'A' } }, display: 'b' } != Concept { codes: { Code { code: '', display: '', version: 'A' } }, display: 'A' })
define D162: (Concept { codes: { Code { code: 'a', display: 'b', version: 'A' } }, display: 'a' } ~ Concept { codes: { Code { code: null, system: null, display: 'A', version: null } }, display: null })
define D163: ({ Code { system: null, display: 'A', version: 'b' } } != { Code { code: 'b', system: 'b', display: 'a', version: 'A' } })
define D164: (Concept { codes: { Code { code: 'A', system: '', display: '' } }, display: 'b' } ~ Concept { codes: { Code { code: '', system: null, display: 'a' } } })
define D165: (Code { code: 'A', display: '', version: 'a' } !~ Code { code: 'A', system: 'b', display: '', version: 'a' })
define D166: ({ Code { code: 'b', system: null, display: 'A', version: 'b' } } != { Code { code: 'A', system: null, display: 'A', version: 'b' } })
define D167: ({ Code { code: 'a', system: 'b', display: 'a', version: null } } = { Code { code: 'b', system: 'A', display: '', version: '' } })
define D168: ((Code { code: null, system: '', display: 'b' }).system is null)
define D169: (Code { code: 'A', system: '', display: 'b', version: 'a' } !~ Code { code: 'A', system: '', display: 'A', version: 'a' })
define D170: (Code { code: 'a', system: null, display: '', version: null }).version
define D171: (Code '' from "CS").code
define D172: (Code { code: '', system: 'A', version: '' } !~ Code { code: 'b', system: 'A', version: '' })
define D173: (Code { system: null, display: null, version: 'a' }).version
define D174: ((Code 'a' from "CS" display 'A').code is null)
define D175: (Code { code: 'b', system: 'a', display: '', version: 'a' } !~ Code { code: 'a', system: 'a', display: '', version: 'a' })
define D176: ((Concept { codes: { Code { code: '', system: 'A', display: null, version: '' } }, display: '' }).display is null)
define D177: (Code { code: '', display: 'a', version: null } ~ Code { code: 'a', system: 'A', version: '' })
define D178: (Concept { codes: { Code { code: 'a', system: 'A', display: '' }, Code { code: 'a', system: null, version: null } }, display: 'a' } ~ Concept { codes: { Code { code: null, system: 'A', display: '' }, Code { code: 'a', system: null, version: null } }, display: 'a' })
define D179: (Code { code: 'a', system: null, display: 'A', version: 'A' }).version
define D180: ("L" != Concept { codes: { Code { system: 'A', display: '' } }, display: 'a' })
define D181: (Concept { codes: { Code { code: 'a', system: null, version: '' }, Code { code: null, system: '', display: 'a', version: 'A' } }, display: 'b' } !~ Concept { codes: { Code { code: 'a', display: 'b', version: 'a' } }, display: '' })
define D182: ((Code { code: 'a', system: 'a', display: 'b' }).version is null)
define D183: (Concept { codes: { Code { code: 'b', system: null, display: 'b', version: '' } }, display: '' } = Concept { codes: { Code '' from "CS" display 'b' }, display: '' })
define D184: ((Code { code: null, display: '' }).display is null)
define D185: ((Code { code: 'a', system: 'A', display: '', version: 'A' }).display is null)
define D186: ((null as Concept) = Concept { codes: { Code { system: 'a', display: 'A', version: '' } }, display: null })
define D187: ((Concept { codes: { Code { code: null, system: 'A', display: '', version: 'A' }, Code { code: 'b', system: '', display: null, version: null } }, display: 'A' }).display is null)
define D188: (Concept { codes: { Code { code: 'a', system: 'a', display: 'b', version: null }, Code { code: null, system: 'b', display: '', version: 'b' } }, display: 'A' } ~ Concept { codes: { Code { code: 'b', system: 'a', display: 'b', version: null }, Code { code: 'A', system: 'b', display: '', version: 'b' } }, display: 'A' })
define D189: (Concept { codes: { Code { code: 'a', system: 'A', display: null, version: 'A' } } } = Concept { codes: { Code { code: 'a', system: '', display: null, version: 'A' } }, display: 'A' })
define D190: (Code { display: '', version: 'A' } = Code { code: 'b', system: 'a', display: 'a', version: 'a' })
define D191: (Concept { codes: { Code { code: 'b', display: 'A' }, Code { code: null, system: 'A', display: 'b' } }, display: 'A' }).display
define D192: (Concept { codes: { Code { code: 'A', system: 'b', display: 'a', version: '' } }, display: 'a' } !~ "K")
define D193: (Code { code: '', system: 'b', display: null } !~ Code { code: 'A', system: '', display: 'b', version: 'A' })
define D194: ({ (null as Code), "X" } = { (null as Code), Code { code: null, system: 'a', display: 'A', version: 'A' } })
define D195: (Code { code: 'b', system: '', display: '', version: null } = Code { code: '', system: null, version: null })
define D196: (Code { code: null, system: '', display: null, version: 'b' } !~ Code { code: '', system: '', display: null, version: 'b' })
define D197: ({ Code { code: null, display: '', version: '' } } = { Code { code: null, display: 'b', version: '' } })
define D198: (Code { code: null, system: '', display: 'a', version: 'A' } !~ Code { system: '', display: '', version: null })
define D199: (Concept { codes: { Code { code: '', system: 'b', display: 'b', version: '' } }, display: 'a' } != (null as Concept))
define D200: ((Concept { codes: { Code { code: null, system: 'b', display: 'b', version: 'a' }, Code { code: '', display: 'b', version: 'b' } }, display: 'a' }).display is null)
define D201: ("X").version
define D202: (Code { system: null, display: '', version: 'b' } = Code { code: null, system: 'a', display: 'b', version: '' })
define D203: (Code { code: 'a', system: '', display: 'a', version: 'A' } != Code { code: 'a', system: '', display: 'a', version: 'a' })
define D204: (Code { code: 'A', system: '', display: 'b', version: 'A' } !~ Code { code: 'A', system: '', display: null, version: null })
define D205: ((Code { code: '', system: null, display: 'A' }).version is null)
define D206: (Code { code: 'b', system: '', version: 'b' } ~ Code { code: null, system: '', display: 'A', version: '' })
define D207: (Code { code: 'A', system: '', display: 'A', version: 'b' } ~ Code { code: 'A', system: '', display: 'A' })
define D208: (Concept { codes: { Code { code: '', display: 'b', version: 'a' }, Code { system: 'b', version: 'A' } }, display: 'b' } !~ Concept { codes: { Code { code: 'b', system: 'b', display: '', version: '' }, Code { code: '', system: 'A', display: 'b', version: 'b' } }, display: null })
define D209: ({ Code { code: null, system: null, display: 'a', version: 'a' } } != { Code { code: null, system: null, display: '', version: 'a' } })
define D210: (Code 'A' from "S" display 'A').code
define D211: (Code '' from "CS").system
define D212: (Code { code: 'b', system: '', display: 'b', version: null } = Code { code: 'A', system: '', display: 'b', version: null })
define D213: ((Code 'a' from "CS" display 'A').version is null)
define D214: (Code { code: 'b', system: 'b', version: 'A' } != "Y")
define D215: (Code { code: 'A', system: 'a', version: '' } ~ Code { code: 'A', system: 'a', display: 'A', version: '' })
define D216: (Concept { codes: { Code { system: 'b', display: null, version: 'a' }, Code { code: 'A', display: 'b', version: 'A' } }, display: 'a' } !~ Concept { codes: { Code { code: '', system: 'a', display: 'A', version: 'b' }, Code { code: null, display: 'b', version: 'A' } }, display: 'a' })
define D217: (Code { code: 'b', system: 'a', version: '' } != Code { code: 'b', system: 'a', version: 'b' })
define D218: ({ Code { code: 'a', system: '', version: null } } = { "X" })
define D219: ("X").system
define D220: (Code { code: 'b', system: '', display: 'b', version: null } ~ Code { code: 'b', system: 'b', display: 'b', version: null })
define D221: (Code { code: 'b', system: 'a', display: '', version: null } !~ Code { code: 'b', system: 'a', display: '', version: 'b' })
define D222: ((Code { system: 'a', display: 'A', version: 'a' }).system is null)
define D223: (Code { code: 'a', system: 'A', display: 'a', version: null } ~ Code { code: 'a', system: 'A', display: 'a', version: null })
define D224: (Code { display: 'A', version: 'a' } ~ Code { code: null, display: 'A', version: 'a' })
define D225: ((null as Code) !~ Code { code: 'A', system: 'b', display: 'b', version: 'a' })
define D226: (Code { code: 'a', system: 'a', display: '' } ~ Code { code: 'b', display: 'A', version: 'A' })
define D227: (Concept { codes: { Code { code: '', system: null, display: 'b' } } } = Concept { codes: { Code { code: 'a', system: null, display: '', version: 'a' } } })
define D228: ({ (null as Code), Code { code: 'A', system: 'A', display: 'A', version: '' } } = { (null as Code), Code { code: '', system: '', display: 'A', version: null } })
define D229: ((Concept { codes: { Code { code: 'b', system: 'a', display: '', version: '' } }, display: 'A' }).display is null)
define D230: (Code { code: null, display: 'b', version: null } ~ Code { code: null, system: 'b', display: 'b', version: null })
define D231: (Concept { codes: { Code { code: 'A', system: 'A', display: 'A' }, Code { code: null, system: '', display: null, version: 'A' } }, display: 'A' } ~ Concept { codes: { Code { code: 'A', system: 'a', display: null, version: 'b' } }, display: '' })
define D232: (Code 'A' from "CS" display 'A').code
define D233: (Code { code: null, system: 'b', display: '' } ~ Code { code: null, system: 'b', display: null })
define D234: (Code { code: 'a', system: 'a', display: '', version: '' } != Code { code: 'a', system: 'a', display: 'a', version: '' })
define D235: (Concept { codes: { Code { code: 'a', display: '', version: 'A' } } } ~ Concept { codes: { Code { code: '', system: null } }, display: 'A' })
define D236: (Concept { codes: { Code { code: 'A', version: 'a' }, Code { code: null, system: 'b', display: 'A' } }, display: null }).display
define D237: (Code { code: null, display: 'A', version: 'b' }).code
define D238: ("Y").version
define D239: ({ (null as Code), Code { code: '', display: 'A', version: 'b' } } = { (null as Code), Code { code: '', display: 'A', version: 'a' } })
define D240: (Concept { codes: { Code { code: 'b', system: 'a', display: null, version: '' } }, display: null }).display
define D241: (Code { system: 'A', display: 'A', version: null } = Code { system: 'a', display: 'A', version: null })
define D242: (Code { code: '', system: 'a', display: 'A', version: 'A' }).version
define D243: (Concept { codes: { Code { code: 'a', system: 'a', display: 'A', version: 'A' } }, display: '' } != Concept { codes: { Code { system: 'a', display: null, version: 'a' } }, display: 'b' })
define D244: (Concept { codes: { Code { code: 'A', system: 'b', display: '' } }, display: 'b' } = Concept { codes: { Code { code: '', version: 'b' } }, display: '' })
define D245: (Code { code: 'a', system: 'a', display: 'A', version: 'A' } = Code { code: 'a', system: '', display: 'A', version: 'A' })
define D246: ((Code '' from "CS").system is null)
define D247: (Concept { codes: { Code { code: '', system: '', version: 'b' }, Code { code: null, system: 'a', display: '', version: 'b' } }, display: 'b' } ~ Concept { codes: { Code { code: '', system: null, display: 'a', version: 'A' }, Code { code: null, system: 'a', display: '', version: 'A' } } })
define D248: ({ Code { code: 'A', system: '', display: '', version: 'A' } } != { Code { code: 'A', system: 'A', display: '', version: 'A' } })
define D249: (Code { code: 'b', system: 'A', display: '' } !~ Code { code: null, system: 'a' })
define D250: (Code { code: 'A', system: 'b', display: 'b', version: 'b' }).version
define D251: ({ (null as Code), Code { code: null, system: '', display: 'a', version: 'A' } } = { (null as Code), Code { code: 'a', system: 'A', display: null, version: '' } })
define D252: (Code { code: 'a', system: 'a', display: 'A', version: 'A' } ~ Code { code: 'a', system: 'a', display: 'A', version: 'a' })
define D253: ((Code { code: '', system: 'a', display: 'A', version: 'a' }).system is null)
define D254: (((null as Code)).system is null)
define D255: (Concept { codes: { Code { code: 'a', system: 'b', display: '', version: 'A' } }, display: null } != Concept { codes: { Code { code: '', system: 'b', display: null, version: 'a' } }, display: 'b' })
define D256: (Code { code: '', display: 'a', version: 'A' } = Code { code: null, system: 'b', display: 'b' })
define D257: (Code 'A' from "CS" display 'b').code
define D258: (Code { code: null, system: 'b' } !~ Code { code: null, system: 'b', display: '' })
define D259: (Concept { codes: { Code { code: null, system: 'b' } }, display: 'A' } = Concept { codes: { Code { code: null, system: 'b', version: '' } }, display: 'A' })
define D260: (Concept { codes: { Code { system: 'b', display: '', version: 'b' } }, display: 'b' } = Concept { codes: { Code { code: '', system: null, display: '' } }, display: 'A' })
define D261: (Code 'A' from "S" display 'A').version
define D262: (Code { code: 'A', system: null, display: 'A', version: 'A' } !~ Code { code: null, system: 'b', display: 'b', version: '' })
define D263: (Code { code: null, system: 'a', display: '', version: 'a' }).version
define D264: (Code { code: 'a', system: '', version: 'a' } != Code { code: 'a', system: null, version: 'a' })
define D265: ((Concept { codes: { Code { system: 'b', display: 'A', version: '' } }, display: 'A' }).display is null)
define D266: (Concept { codes: { Code { system: '', display: null, version: '' } }, display: 'a' } = Concept { codes: { Code { code: 'a', system: '', display: null, version: '' } }, display: 'a' })
define D267: ((Code 'A' from "S").version is null)
define D268: ((Code 'A' from "S" display '').version is null)
define D269: ({ (null as Code), Code { code: 'b', system: '', version: null } } = { (null as Code), Code { code: 'A', system: '', version: null } })
define D270: ((Concept { codes: { "Z", Code { code: 'b', display: 'a', version: 'b' } }, display: 'b' }).display is null)
define D271: (Concept { codes: { Code { code: '', system: null, display: 'A', version: null }, Code { code: 'A', system: '', display: 'b', version: null } }, display: '' } !~ "L")
define D272: (Code { code: '', system: 'a', display: 'a', version: 'b' } = Code { code: '', system: 'a', display: 'a', version: '' })
define D273: (Concept { codes: { Code { system: null, version: 'A' } }, display: '' } != Concept { codes: { Code { code: '', system: null, version: 'A' } } })
define D274: (Code { system: '', display: 'A', version: 'a' } !~ Code { code: '', system: '', display: 'A', version: 'a' })
define D275: ((Code { code: null, system: 'a', display: 'a', version: 'A' }).code is null)
define D276: (Code '' from "S").version
define D277: (Code { code: 'A', system: 'b', display: 'A', version: 'b' } = Code { code: 'A', system: 'b', display: 'a', version: 'b' })
define D278: ((Code 'A' from "S" display 'b').system is null)
define D279: (Concept { codes: { Code { system: 'A', version: 'a' }, Code { code: 'A', display: 'b', version: '' } }, display: '' }).display
define D280: (Code 'A' from "S" display 'A').display
define D281: ({ Code 'a' from "CS" display 'b' } != { Code { code: '', system: 'a' } })
define D282: (Code { code: '', system: 'A', display: 'b', version: 'a' } !~ Code { code: '', system: 'A', version: 'a' })
define D283: (Code { code: 'b', system: '', display: null } != Code { code: null, version: 'a' })
define D284: (Code { code: 'A', system: 'b', display: null, version: 'A' } != Code { code: 'A', system: 'b', display: '', version: 'A' })
define D285: (Concept { codes: { Code { code: 'A', system: null, display: 'b', version: 'A' } }, display: null } = (null as Concept))
define D286: ((null as Concept) ~ Concept { codes: { Code { code: '', system: null, display: 'a', version: '' }, Code { code: 'a', display: '', version: 'A' } }, display: null })
define D287: (Concept { codes: { Code { code: '', system: 'b', display: null, version: 'a' }, Code { code: '', system: 'A', display: '' } }, display: 'a' }).display
define D288: ("Y").display
define D289: (Code { code: '', system: '', display: 'a', version: '' } != Code { code: '', system: 'b' })
define D290: (("Y").display is null)
define D291: ({ Code { system: '', display: '', version: null } } = { Code { system: '', display: '', version: '' } })
define D292: (Code { code: 'b', system: 'A', display: 'A', version: 'a' } ~ Code { code: 'b', display: 'A', version: 'a' })
define D293: (Code { code: 'b', display: null }).version
define D294: (Concept { codes: { Code { code: 'a', system: 'A', version: '' } }, display: 'b' } != "K")
define D295: (Code { code: null, system: 'A', display: null, version: 'A' } ~ Code { code: null, display: null, version: 'A' })
define D296: (Concept { codes: { Code { code: 'a', system: 'a', display: 'b', version: 'b' } }, display: 'a' } ~ Concept { codes: { Code { code: '', display: 'A', version: 'A' }, Code { code: 'a', system: 'a', version: '' } } })
define D297: ((Concept { codes: { Code { code: 'a', display: null, version: 'b' } }, display: '' }).display is null)
define D298: ((Concept { codes: { Code { code: 'b', system: 'b', display: null, version: 'A' } }, display: 'b' }).display is null)
define D299: (Code { code: 'a', display: 'a', version: 'b' } !~ Code { code: 'a', display: 'a', version: 'b' })
define D300: (Code { code: '', system: '', display: 'b', version: 'A' } ~ Code { code: '', system: null, display: 'b' })
define D301: (Code { code: null, system: 'b', display: '', version: 'A' } = Code { code: 'A', system: 'b', display: '', version: 'A' })
define D302: ({ (null as Code), Code { display: 'b', version: 'b' } } = { (null as Code), Code { code: 'b', display: '', version: 'a' } })
define D303: ((Concept { codes: { Code { code: 'b', system: null, display: null, version: null }, Code { system: 'a', display: null, version: 'A' } }, display: null }).display is null)
define D304: (Code { code: 'A', version: 'A' } ~ Code { code: 'a', system: 'a', display: 'A' })
define D305: ("X" != Code { code: 'a', display: 'b', version: 'b' })
define D306: (Code { code: '', system: 'b', version: 'b' } !~ Code { code: 'b', display: null, version: '' })
define D307: (Concept { codes: { "X", Code { system: 'b', display: 'b' } }, display: 'b' }).display
define D308: (Code { system: null, display: 'b', version: 'A' } != Code { code: 'A', system: '', display: 'A', version: 'a' })
define D309: ((Code { code: 'A', display: 'A', version: 'A' }).code is null)
define D310: (Code '' from "S" display 'a' = Code { code: null, system: 'b', display: '', version: 'b' })
define D311: (Code { code: 'b', system: 'a', version: '' } !~ Code { code: 'b', system: 'a', display: null, version: '' })
define D312: (((null as Code)).code is null)
define D313: ("Z").system
define D314: (Code 'A' from "S" display 'a').system
define D315: (Concept { codes: { Code 'A' from "S" display 'a', Code { code: 'A', system: '', display: '', version: 'a' } } }).display
define D316: (Code { system: 'A', display: 'b', version: 'A' }).system
define D317: ("X").display
define D318: (Concept { codes: { Code { code: 'b', system: 'A', version: 'a' } }, display: 'a' }).display
define D319: (Code { code: 'b', system: 'A', display: '', version: 'b' } !~ Code { system: 'A', display: '', version: '' })
define D320: ((Concept { codes: { Code { code: 'A', display: 'b', version: 'a' }, Code { system: 'b', display: 'A' } }, display: null }).display is null)
define D321: (Concept { codes: { "Y" } } = Concept { codes: { Code { code: 'a', version: 'A' } }, display: 'a' })
define D322: ((Concept { codes: { Code { code: 'a', system: '', display: null, version: null }, Code { code: null, system: 'b' } }, display: '' }).display is null)
define D323: ("Z").code
define D324: ({ Code { code: 'b', system: 'b', display: 'a', version: 'A' } } = { Code { code: 'b', system: 'A', display: 'b', version: null } })
define D325: (Code { code: 'A', system: 'A', display: 'b', version: null } = Code { code: null, system: 'A', display: 'b', version: null })
define D326: (Code 'A' from "S").display
define D327: (Code 'a' from "CS" display 'A').version
define D328: (Code 'a' from "S" display '').system
define D329: (Concept { codes: { Code { code: null, system: 'A', display: 'a' } }, display: 'b' }).display
define D330: (Code { version: 'b' } != Code { system: 'A', version: 'b' })
define D331: ((Code { code: 'b', system: null, display: 'a', version: 'b' }).display is null)
define D332: (Concept { codes: { Code { code: 'a', system: '', display: 'a', version: null } }, display: 'A' }).display
define D333: (Code { code: '', display: '', version: null } ~ Code { code: '', display: '', version: 'a' })
define D334: (Concept { codes: { Code { code: '', display: '', version: 'A' }, Code { system: '', display: '', version: 'A' } } }).display
define D335: ((Concept { codes: { Code { code: '', system: 'a', display: null, version: 'b' }, Code { code: 'A', system: null, display: null, version: '' } }, display: 'b' }).display is null)
define D336: (Code { code: 'b', system: 'b', display: null, version: null }).version
define D337: ((Concept { codes: { Code { code: 'A', system: '', display: 'a', version: '' } }, display: '' }).display is null)
define D338: (Code { system: 'A', display: null, version: null } !~ Code { code: '', system: 'A', display: null, version: null })
define D339: (Code { code: 'b', display: 'a', version: 'A' }).display
define D340: (Code { code: '', display: 'b', version: 'b' } !~ Code { code: 'a', system: null, display: '', version: null })
define D341: ((Code { code: 'b', system: null, version: '' }).display is null)
define D342: ((Code { code: '', system: 'a' }).display is null)
define D343: (Concept { codes: { Code { system: null }, Code { code: '', system: 'b', version: '' } }, display: 'A' } ~ Concept { codes: { Code { system: null, version: null }, Code { code: '', system: 'b', version: 'a' } }, display: null })
define D344: (Code { system: null, display: 'A', version: 'b' }).display
define D345: (Code { code: 'A', system: null, display: 'a', version: 'A' } = Code { code: 'a', system: 'b', display: 'a', version: null })
define D346: (Code { code: null, system: 'a', display: '', version: 'A' } ~ Code { code: null, system: null, display: '', version: 'A' })
define D347: ("Z").display
define D348: (Concept { codes: { Code { code: null, display: 'a', version: 'b' }, Code { code: '', system: 'a', display: '', version: 'a' } }, display: 'a' } ~ Concept { codes: { Code { code: '', system: 'b', display: null }, Code { code: '', system: 'A', display: '', version: 'a' } }, display: 'b' })
define D349: (Code { code: '', display: 'a', version: 'b' }).version
define D350: ((Code 'a' from "CS" display '').version is null)
define D351: (Concept { codes: { Code { system: 'A', display: 'b', version: 'b' }, Code { code: null, system: 'b', display: null, version: '' } } } !~ Concept { codes: { Code { system: 'A', display: '', version: 'b' }, Code { code: null, system: 'b', display: 'a', version: '' } } })
define D352: (Concept { codes: { Code { code: 'a', system: 'A', version: null } }, display: null } ~ Concept { codes: { Code { code: 'a', system: 'b', display: 'A', version: 'b' }, Code { system: 'A', version: null } }, display: 'a' })
define D353: (Concept { codes: { Code { code: null, system: 'A', display: 'b' } }, display: 'A' }).display
define D354: (Concept { codes: { Code { system: '', display: 'a', version: 'a' } }, display: null } !~ Concept { codes: { Code { display: '', version: 'a' } }, display: 'A' })
define D355: (Code '' from "S" display '').display
define D356: (("Y").code is null)
define D357: (Code 'a' from "S" display 'A').version
define D358: ("Z").version
define D359: (Concept { codes: { "Z" } }).display
define D360: (("Y").system is null)
define D361: ({ Code { system: 'b', display: 'A', version: '' } } != { Code { display: 'A', version: '' } })
define D362: (Concept { codes: { Code { code: 'A', display: null, version: 'a' } }, display: null } = Concept { codes: { Code { code: 'A', display: null, version: null } }, display: null })
define D363: ((Concept { codes: { Code { code: 'b', system: 'a', display: 'b', version: null } }, display: '' }).display is null)
define D364: ((Code { code: 'a', system: 'A', version: 'a' }).display is null)
define D365: (Code { code: null, system: '', display: '', version: 'b' }).system
define D366: (Code { code: null, system: '', version: '' } !~ Code { code: null, system: '', display: 'A', version: '' })
define D367: ((Concept { codes: { Code { code: 'b', system: '', display: 'a' } } }).display is null)
define D368: (Code { code: '', system: '', display: 'a', version: 'b' } ~ Code { code: '', system: 'a', display: 'b', version: 'A' })
define D369: (Code { code: 'b', system: 'A', version: 'a' } = Code { code: 'b', system: 'b', version: 'a' })
define D370: (Concept { codes: { Code { system: 'b', display: 'a', version: 'b' } }, display: 'a' }).display
define D371: (Code { code: 'a', system: null, display: null } !~ Code { code: '', system: 'A', display: 'a', version: 'A' })
define D372: ((Concept { codes: { Code { system: 'A', display: '', version: 'A' }, Code { system: null, display: 'A', version: 'a' } }, display: 'a' }).display is null)
define D373: ({ (null as Code), Code { code: 'A', system: 'a', display: 'A', version: 'A' } } = { (null as Code), Code { code: 'a', display: 'A', version: 'A' } })
define D374: ({ (null as Code), Code { code: 'b', system: 'a', display: '', version: 'A' } } = { (null as Code), Code { code: 'A', display: 'A', version: 'b' } })
define D375: (Code { code: '', system: '', version: 'A' } ~ Code { code: null, system: '', display: 'b', version: 'A' })
define D376: (Concept { codes: { Code { code: '', system: 'A', version: 'b' }, Code { system: null, display: null, version: 'A' } } } ~ Concept { codes: { Code { code: '', system: 'A', display: 'b', version: 'A' }, Code { system: null, version: 'A' } } })
define D377: ({ Code { code: 'a', system: 'A', display: '', version: 'a' } } = { Code { code: 'a', display: '', version: 'a' } })
define D378: (("Z").system is null)
define D379: ((Concept { codes: { Code { code: 'b', display: null, version: 'A' }, Code { system: 'b', display: '', version: 'a' } }, display: '' }).display is null)
define D380: ({ (null as Code), Code { code: 'b', display: null, version: null } } != { (null as Code), Code { code: 'b', system: '', display: null, version: null } })
define D381: (Code { code: 'b', system: 'a', display: '', version: '' } = Code { code: 'b', system: 'a', display: '', version: 'A' })
define D382: ((Code { system: '', display: 'A' }).code is null)
define D383: ((Concept { codes: { Code { code: '', system: 'A', display: 'a', version: 'A' } }, display: 'b' }).display is null)
define D384: ((Code '' from "CS" display 'b').version is null)
define D385: (Code { system: '', display: '', version: null } ~ Code { code: 'b', system: '', display: '', version: null })
define D386: ({ (null as Code), Code { code: 'A', system: 'b', display: '', version: null } } = { (null as Code), Code { code: 'A', system: 'b', version: null } })
define D387: (Code 'a' from "S").version
define D388: ((Concept { codes: { Code { code: 'b', system: 'b', display: '', version: 'A' } }, display: 'a' }).display is null)
define D389: (Code { code: 'b', system: 'b', display: 'A', version: 'a' } != Code { code: 'b', system: '', display: 'A', version: 'a' })
define D390: (Concept { codes: { Code { code: '', system: 'A', display: 'b', version: null }, Code { code: null, system: 'a', display: 'b', version: 'A' } }, display: 'a' } !~ Concept { codes: { Code { code: 'b', system: null, version: 'A' }, Code { code: 'a', system: '', version: null } }, display: 'a' })
define D391: (Concept { codes: { Code { code: '', system: '', display: 'b', version: 'b' } }, display: 'a' } != Concept { codes: { Code { code: '', system: '', display: 'b', version: 'b' } }, display: 'a' })
define D392: (Code { code: 'a', system: '', display: 'b', version: 'a' } !~ Code { code: 'a', system: '', display: 'b', version: 'a' })
define D393: ((Code { code: 'a', system: null, display: null, version: 'a' }).display is null)
define D394: (Concept { codes: { Code { code: null, system: 'A', version: '' } }, display: 'a' }).display
define D395: ({ (null as Code), Code { code: 'a', system: 'a', display: 'b', version: 'A' } } != { (null as Code), Code { code: 'a', system: 'A', display: 'b', version: 'A' } })
define D396: (Code { code: 'a', system: '', display: 'A', version: 'a' } != Code { code: 'a', system: 'A', display: 'A', version: 'a' })
define D397: (Code { code: null, system: 'A', display: 'A', version: '' }).system
define D398: ({ Code { code: 'b', system: null, display: 'a', version: 'A' } } = { Code { code: null, display: null, version: null } })
define D399: (Code { code: 'A', system: '', display: 'A', version: null } !~ Code 'A' from "CS" display 'b')
define D400: ({ Code { code: null, system: 'a', display: null, version: 'b' } } != { Code { code: 'a', system: null, display: null, version: null } })
define D401: (Code { system: 'b', display: 'a', version: 'a' } !~ Code { code: 'a', system: 'A', display: 'a', version: 'b' })
define D402: ({ Code { system: 'a', display: 'A', version: 'b' } } = { Code { code: 'A', system: 'b', display: 'a', version: 'A' } })
define D403: ((Concept { codes: { "X", Code { code: '', system: '', display: 'a' } } }).display is null)
define D404: (("L").display is null)
define D405: ({ Code { code: null, display: 'a', version: null } } = { Code { code: null, display: 'a', version: '' } })
define D406: (Code { code: null, display: null, version: 'b' } ~ Code { code: null, display: null, version: 'b' })
define D407: (Code { code: 'a', system: 'b', display: '', version: 'A' } = Code { code: 'a', display: '', version: 'A' })
define D408: ((Code { code: null, display: 'A', version: 'A' }).code is null)
define D409: (Code { code: 'a', system: 'A', display: null, version: '' } ~ Code { code: 'a', system: 'A', display: null, version: 'b' })
define D410: ((Code '' from "S").code is null)
define D411: (Code { system: null, display: 'A', version: 'a' } = Code { system: 'a', display: 'b', version: 'a' })
define D412: (Code { code: 'b', system: 'a', display: 'A', version: 'b' } = Code { code: null, display: '', version: '' })
define D413: (("X").version is null)
define D414: (Concept { codes: { Code '' from "CS" }, display: 'b' } != Concept { codes: { Code { system: 'a', display: 'b', version: null } }, display: 'b' })
define D415: (Code { code: '', system: 'A', display: 'b', version: 'A' } !~ Code { code: 'A', display: '', version: 'A' })
define D416: (("Z").version is null)
define D417: ({ Code { code: 'b', system: 'b', display: 'b', version: 'b' } } = { Code { code: 'b', system: 'b', display: 'b', version: 'b' } })
define D418: ({ Code { code: null, system: null, display: 'a', version: 'a' } } = { Code { code: 'A', display: '', version: 'A' } })
define D419: ((Concept { codes: { Code { code: '', system: 'a', display: 'A', version: 'a' }, Code { code: '', system: 'b', display: '', version: '' } }, display: null }).display is null)
define D420: (Code { code: 'b', system: '', display: 'b', version: 'A' } = Code { code: null, version: 'a' })
define D421: ({ (null as Code), Code { code: 'b', system: 'A', display: 'A', version: 'a' } } = { (null as Code), Code { system: 'A', display: 'A', version: 'a' } })
define D422: ((Code 'a' from "CS" display 'b').system is null)
define D423: ((Code { code: 'A', system: 'b', display: '' }).system is null)
define D424: ((null as Concept) != Concept { codes: { Code { code: null, display: 'A' } }, display: '' })
define D425: ((Code { system: 'a', version: 'A' }).version is null)
define D426: (Code { code: null, system: '', display: 'A', version: 'a' } ~ Code { code: 'A', system: '', display: 'A', version: 'a' })
define D427: (Code { system: 'a', version: 'A' } !~ Code { code: 'a', system: 'a', display: null, version: 'A' })
define D428: ((Code { code: null, system: null, version: '' }).system is null)
define D429: ((Code '' from "S").display is null)
define D430: ({ Code { code: 'a', display: null, version: 'b' } } != { Code { system: '', display: null, version: null } })
define D431: (((null as Code)).display is null)
define D432: (Code { code: 'b', system: 'A', display: '', version: 'b' }).system
define D433: (Code { code: '', system: null, display: '', version: 'a' } !~ Code { system: null, display: '', version: 'a' })
define D434: ({ Code { code: '', system: 'a', display: null } } = { Code { code: 'b', system: 'A', version: 'b' } })
define D435: (Code { code: 'a', display: 'b' }).system
define D436: (Concept { codes: { Code { code: null, display: 'a', version: 'a' }, Code 'A' from "S" }, display: 'A' }).display
define D437: ({ Code { system: '', display: '', version: null } } = { Code { system: '', display: null, version: null } })
define D438: (Concept { codes: { Code { code: null, system: '', display: 'a' } }, display: 'A' } != Concept { codes: { Code { code: '' } }, display: null })
define D439: (Code { code: null, system: '', version: '' }).code
define D440: ({ (null as Code), Code { code: 'b', system: 'a', display: '' } } = { (null as Code), Code { code: 'a', display: 'a' } })
define D441: ({ Code { system: 'b', display: 'a', version: '' } } = { Code { system: null, display: 'a', version: '' } })
define D442: (Concept { codes: { Code { code: 'b', system: 'A', display: 'a', version: null } }, display: 'b' } = Concept { codes: { Code { code: 'b', system: '', display: 'a', version: null } }, display: 'a' })
define D443: (Concept { codes: { Code { code: '', system: 'b', display: 'a', version: '' } }, display: 'b' } !~ Concept { codes: { Code { code: 'A', display: 'b', version: null }, Code 'A' from "S" } })
define D444: ("L" !~ (null as Concept))
define D445: (Concept { codes: { Code { code: 'A', system: 'b', display: 'b', version: 'b' } }, display: 'A' } != Concept { codes: { Code { system: '', version: 'b' } }, display: 'A' })
define D446: (Concept { codes: { Code { code: null, system: 'a', display: 'a', version: '' } }, display: 'a' } != Concept { codes: { Code { code: 'A', system: 'a', display: 'a', version: '' } } })
define D447: (Concept { codes: { Code { code: null, system: null, display: '', version: 'A' } }, display: 'b' } ~ Concept { codes: { Code { code: 'a', system: 'A', version: 'a' } }, display: 'b' })
define D448: (Code { code: 'b', system: 'A', display: null, version: 'A' } = Code { code: 'b', system: 'A', display: '', version: 'A' })
define D449: (Code { code: 'b', system: 'A', display: 'a' } = Code { code: 'b', system: 'A', display: 'a' })
define D450: (Concept { codes: { Code { code: 'a', system: '', display: 'b', version: null } }, display: null } = Concept { codes: { Code { code: 'A', system: 'a', display: '', version: 'A' } }, display: null })
define D451: ((Code { code: 'a', display: null, version: 'a' }).system is null)
define D452: ((Concept { codes: { Code { code: null, display: '', version: 'a' }, Code { code: '', system: 'b', display: '', version: 'b' } } }).display is null)
define D453: (Code { code: 'b', system: '', display: '' } != Code { code: 'b', system: '', display: '' })
define D454: (Code { code: 'A', display: '', version: '' } = Code { code: null, display: '', version: '' })
define D455: (Code { code: 'b', system: 'b', display: 'b', version: '' } ~ Code { code: '', system: 'b', display: 'b', version: '' })
define D456: (Code { code: 'A', system: 'a' }).code
define D457: (Code { code: 'a', display: null, version: 'A' } != Code { code: '', system: 'b', display: 'b' })
define D458: ({ Code { system: '', display: null, version: '' } } != { Code { system: 'A', display: null, version: '' } })
define D459: (Concept { codes: { Code { code: 'A' } } } = Concept { codes: { Code { code: null, system: 'A' } }, display: 'b' })
define D460: ((Code { code: 'a', system: 'b', display: 'b' }).system is null)
define D461: (Concept { codes: { Code { code: 'A', system: 'a', display: 'A', version: null }, Code { code: 'a', system: null, display: 'A' } }, display: 'b' }).display
define D462: (Concept { codes: { Code { code: 'a', system: null, display: 'a', version: null } }, display: 'a' }).display
define D463: (Code { code: 'A', system: 'A', display: 'a', version: null }).system
define D464: ((Code 'a' from "CS").system is null)
define D465: (Concept { codes: { Code 'A' from "CS" display 'A', Code { code: '', system: 'A', display: 'b', version: 'A' } }, display: 'b' }).display
define D466: ((null as Concept) ~ Concept { codes: { Code { system: 'A', display: 'b', version: '' } } })
define D467: (Code { code: '', system: 'b', display: '', version: 'b' } ~ Code { code: '', system: 'b', display: '', version: 'A' })
define D468: (Concept { codes: { Code { code: 'A', system: 'a', display: 'a', version: 'b' } }, display: 'A' } = Concept { codes: { Code { code: 'a', system: null, display: 'b', version: 'b' } }, display: 'A' })
define D469: (Code { code: 'A', system: 'A', display: 'a' } = Code { code: null, system: 'A', display: 'a' })
define D470: (Code { code: 'a', system: 'a', display: 'A', version: '' } !~ Code { code: 'a', display: 'A', version: 'A' })
define D471: ((Concept { codes: { Code { code: 'A', system: null, display: null, version: 'a' } }, display: 'b' }).display is null)
define D472: ((Code { system: 'b', display: 'a' }).system is null)
define D473: ({ Code { code: 'b', system: null, display: 'A', version: 'b' } } != { Code 'A' from "S" display '' })
define D474: (Code { code: 'a', system: null, display: 'b', version: null }).system
define D475: (Code { code: 'A', system: 'a', display: '', version: '' } = Code { code: 'A', system: 'b', version: 'b' })
define D476: (Concept { codes: { Code { system: 'a', display: null, version: 'b' } }, display: '' }).display
define D477: ({ (null as Code), Code { code: 'A', system: '', display: '' } } = { (null as Code), Code { code: 'b', system: 'a', version: null } })
define D478: (Code { code: 'b', system: 'a', display: 'a', version: 'A' }).system
define D479: (Code { code: 'A', system: 'A', version: 'A' } = Code { code: 'A', system: 'A', version: 'A' })
define D480: ((Concept { codes: { Code { code: 'a', system: 'A', display: null, version: 'A' }, Code { code: 'A', system: '', display: null } }, display: null }).display is null)
define D481: ((Concept { codes: { Code { code: '', display: null, version: null } }, display: null }).display is null)
define D482: ({ Code { display: null } } != { Code { system: 'b', display: 'A', version: 'A' } })
define D483: ((null as Code) ~ Code { code: 'A', system: null, display: 'a', version: 'a' })
define D484: (Code { code: null, display: 'A', version: 'A' } !~ Code { system: 'b', version: 'A' })
define D485: (Code { code: 'A', system: 'A', display: '', version: 'b' } ~ Code { code: 'A', system: '', display: '', version: 'b' })
define D486: ((Code '' from "S" display 'b').version is null)
define D487: (Concept { codes: { Code { code: '', system: 'a', display: null, version: null } }, display: 'A' }).display
define D488: (Code { code: 'b', display: 'A', version: 'a' } !~ Code { code: 'b', display: 'A', version: 'b' })
define D489: ({ Code { code: 'b', system: 'a', display: 'A' } } = { Code { system: 'A', display: 'a', version: 'a' } })
define D490: (Code { code: 'a', system: '', version: '' } = Code { code: 'a', system: '', version: '' })
define D491: (Code { code: 'A', system: 'b', display: null, version: '' }).version
define D492: (Code { code: 'b', system: 'A', display: null, version: 'A' } ~ Code { code: 'b', system: 'A', display: null, version: 'a' })
define D493: (Concept { codes: { Code { code: null, system: null, display: 'a', version: 'a' } }, display: null } = "L")
define D494: (Code { code: 'A', system: 'a', version: '' } !~ Code { system: 'A', display: '', version: 'A' })
define D495: (Code { code: 'b', system: '', display: null, version: 'a' } != Code { code: 'b', system: '', display: null, version: 'b' })
define D496: (Concept { codes: { Code { code: '', version: 'a' } }, display: 'A' } != "L")
define D497: ({ Code { code: null, system: 'A', version: 'a' } } = { Code { code: null, system: 'b', display: 'A', version: 'A' } })
define D498: (Code { code: 'b', system: 'a', display: 'A', version: 'A' } != Code { system: 'A', version: '' })
define D499: (Code { code: 'a', display: null, version: 'A' } !~ Code { code: 'a', system: 'A', display: 'b', version: 'a' })
define D500: (Concept { codes: { Code { system: 'b', display: null, version: null } }, display: 'b' } = Concept { codes: { Code { system: '', display: null, version: null } }, display: 'b' })
define D501: (Concept { codes: { Code { system: null, display: 'a', version: 'b' } }, display: 'a' } != Concept { codes: { Code { version: 'b' } }, display: 'a' })
define D502: ((Code { code: '', version: null }).version is null)
define D503: ({ Code { system: 'A', display: 'b', version: 'b' } } = { Code { system: 'A', display: 'b', version: 'b' } })
define D504: ({ (null as Code), Code { code: 'b', system: 'A', display: null, version: '' } } = { (null as Code), Code { code: '', system: 'A', display: null, version: '' } })
define D505: (Code { system: 'A', display: null, version: '' } = Code { system: 'a', display: null, version: '' })
define D506: ((Code { code: '', system: 'b', display: 'A', version: null }).version is null)
define D507: ((Code 'A' from "CS" display 'A').code is null)
define D508: (Code 'A' from "S" display '').system
define D509: ((Concept { codes: { Code { system: 'A', display: null, version: '' }, Code { code: 'A', version: 'b' } }, display: '' }).display is null)
define D510: ((null as Concept) ~ Concept { codes: { Code { system: 'b', display: 'b', version: 'A' }, Code { code: null, system: '', display: 'a', version: 'A' } }, display: 'a' })
define D511: ("X").code
define D512: (Code { code: 'a', system: '', display: 'a', version: 'a' } != Code { code: '', system: '', display: 'a', version: 'a' })
define D513: ("Z" != Code { code: 'a', system: 'b', version: 'a' })
define D514: (Code { code: 'a', system: 'a', display: 'b', version: 'b' }).display
define D515: ({ Code { system: 'b', display: 'a', version: 'A' } } != { Code { system: 'b', display: 'a', version: null } })
define D516: ({ (null as Code), Code { code: '', system: 'A', version: null } } != { (null as Code), Code { code: '', system: '', version: null } })
define D517: (Code { code: 'a', system: '', display: 'A' } ~ Code { code: 'a', version: null })
define D518: (Concept { codes: { Code { display: '', version: '' }, Code { system: 'a', display: 'A', version: 'a' } }, display: 'A' } ~ Concept { codes: { Code { code: '', system: 'b', display: null, version: 'a' } }, display: 'a' })
define D519: (Code { system: 'A', version: 'A' } ~ Code { system: 'b', display: '', version: '' })
define D520: ({ (null as Code), Code { system: 'A', display: 'b', version: 'A' } } = { (null as Code), Code { code: 'b', system: '', version: '' } })
define D521: (Code { code: 'b', system: '', display: '' } ~ Code { code: 'b', system: 'b', display: 'A', version: 'a' })
define D522: (Code { code: '', system: null, display: null, version: 'A' } = Code { code: '', system: null, version: 'A' })
define D523: ({ Code { code: null, system: 'A', display: null, version: 'b' } } = { Code { code: null, system: 'A', display: null, version: 'a' } })
define D524: (Code '' from "S" display 'A').display
define D525: (Concept { codes: { Code { code: '', system: null, display: 'b', version: 'a' } }, display: 'a' }).display
define D526: (Code { code: 'A', system: null, display: 'b', version: '' } != Code { code: 'A', system: 'b', display: 'b', version: '' })
define D527: (Code { code: null, system: null, version: 'A' } = Code { code: null, version: 'A' })
define D528: (Code { code: 'b', system: '', display: 'b', version: 'A' }).version
define D529: (Code { code: '', system: null, display: 'a', version: null }).system
define D530: (Code { code: '', system: 'A', display: 'A', version: 'a' } !~ Code { code: 'a', system: 'a', display: null, version: 'b' })
define D531: (Code { system: 'b' }).display
define D532: (Code '' from "CS" display 'b').version
define D533: ({ (null as Code), Code { code: 'b', system: '', display: 'A', version: 'a' } } = { (null as Code), Code { code: 'b', system: null, display: 'A', version: 'a' } })
define D534: ({ (null as Code), Code { code: 'b', system: null, display: null, version: '' } } != { (null as Code), Code { code: 'A', system: '', display: '', version: 'a' } })
define D535: (Concept { codes: { Code { code: 'b', system: '', display: '', version: '' }, Code { code: 'b', system: null, display: 'A', version: 'a' } } }).display
define D536: ((Code 'A' from "CS" display '').display is null)
define D537: ((Concept { codes: { Code { code: 'A', system: null, display: null, version: 'b' } } }).display is null)
define D538: (Concept { codes: { Code { code: 'b', system: 'A', version: 'A' }, Code { code: 'A', system: 'b', display: null, version: 'A' } } }).display
define D539: ((Code { code: null, system: '', display: 'A' }).display is null)
define D540: ((Code 'a' from "S" display 'a').code is null)
define D541: (Code { code: 'A', system: 'b', version: 'b' }).version
define D542: (Code { code: 'a', version: 'b' } != Code { code: '', version: 'b' })
define D543: ((Code { code: null, system: 'b', display: 'a', version: '' }).system is null)
define D544: (Code { code: 'b', system: 'a', display: 'a', version: null }).display
define D545: ({ Code { code: 'a', system: '', version: 'a' } } = { Code { code: 'A', system: 'a', display: 'A', version: null } })
define D546: ({ Code { code: 'b', system: 'A', display: 'b', version: 'a' } } = { Code { code: 'A', display: 'A', version: null } })
define D547: ((Code 'A' from "CS" display 'b').display is null)
define D548: (Code { code: 'a', system: '', display: '', version: '' } ~ Code { code: 'a', system: '', display: '', version: 'A' })
define D549: (Concept { codes: { Code { code: 'A', system: 'A', version: null } }, display: '' } = Concept { codes: { Code { code: 'A', system: 'A', version: null } }, display: '' })
define D550: ({ Code { code: null, system: 'A', display: 'A', version: 'a' } } != { Code { code: 'b', system: 'A', display: 'A', version: 'a' } })
define D551: ({ Code { code: null, display: 'a', version: '' } } = { Code { code: null, display: 'a', version: '' } })
define D552: ("K" ~ Concept { codes: { "X" }, display: 'b' })
define D553: (Concept { codes: { Code { code: null, system: 'a', display: 'a', version: 'a' } }, display: null } != Concept { codes: { Code { code: null, system: 'A', display: 'b', version: null } } })
define D554: (Concept { codes: { Code { code: 'A', system: 'a', display: 'A' }, Code { code: '', system: 'b', display: 'A' } }, display: 'b' } ~ Concept { codes: { Code 'A' from "CS" display 'a', Code { code: 'A', system: null, display: 'b', version: 'A' } }, display: 'A' })
define D555: (Code { code: 'b', system: null, display: 'A', version: 'A' }).version
define D556: (Concept { codes: { Code { system: null, display: 'a' } }, display: '' }).display
define D557: (Concept { codes: { Code { code: 'a', system: null, display: 'a', version: 'a' }, Code { code: 'A', system: 'a', display: '' } }, display: 'A' } !~ Concept { codes: { Code { code: 'b', system: '', version: 'a' }, Code { code: 'A', system: 'a', display: '', version: 'A' } }, display: 'A' })
define D558: ((Concept { codes: { Code { system: '', display: '', version: 'a' }, "Z" }, display: 'A' }).display is null)
define D559: (Code { system: 'b', display: 'A' } !~ Code { code: 'a', display: null, version: null })
define D560: (Concept { codes: { Code { code: 'b', system: '', display: '' } } } = Concept { codes: { Code { code: 'A', system: null, display: 'a', version: '' } } })
define D561: (Concept { codes: { Code { code: 'a', system: 'b', display: '', version: 'a' } }, display: 'a' }).display
define D562: (Concept { codes: { Code { code: 'A', system: '', display: '', version: 'A' } }, display: 'A' } ~ Concept { codes: { Code { code: '', system: 'A', display: 'a', version: 'b' }, Code { system: '', display: '' } }, display: null })
define D563: ({ Code { code: 'a', system: 'a' } } = { Code { code: 'a', display: 'a', version: null } })
define D564: ({ Code { code: 'A', system: null, display: 'a', version: 'b' } } != { Code { code: 'A', system: null, display: 'a', version: 'a' } })
define D565: (Code { code: 'A', system: null, display: 'a', version: 'b' } != Code { code: null, system: 'a' })
define D566: (Code { code: null, system: '', display: 'a', version: 'a' } = Code { code: null, system: '', display: 'A', version: 'a' })
define D567: (Concept { codes: { Code { code: '', system: 'b', display: 'a', version: 'b' } }, display: '' } != Concept { codes: { Code { system: 'b', display: 'A', version: 'A' } } })
define D568: (Concept { codes: { Code { system: 'a', display: '', version: null }, Code { code: 'A', display: null, version: 'A' } }, display: null }).display
define D569: (Concept { codes: { Code { code: 'A', display: 'a', version: null }, Code { code: '', system: 'A', display: null, version: '' } } }).display
define D570: (Code { system: 'a', version: 'a' } ~ Code { code: 'b', system: null, display: 'a', version: 'A' })
define D571: (Code { code: '', system: '', version: null } != Code { code: '', system: '', display: 'A', version: null })
define D572: (Code { code: 'b', display: null } = Code { code: 'b', display: null })
define D573: ((Concept { codes: { Code { version: '' }, Code { code: '', system: '', display: '', version: 'b' } }, display: 'A' }).display is null)
define D574: (Code { display: 'b', version: null }).system
define D575: (Code { code: '', version: 'a' } ~ Code { code: '', display: 'b', version: 'a' })
define D576: ((Concept { codes: { Code { code: 'a', system: 'b', display: 'b', version: 'a' }, "Y" }, display: '' }).display is null)
define D577: (Code { code: 'A', system: 'A', display: 'a', version: 'A' }).system
define D578: (Code { code: '', system: 'a', display: 'a', version: null } ~ Code { code: 'a', system: '', display: 'a', version: 'b' })
define D579: ({ Code { code: 'b', system: 'a', display: '', version: '' } } = { Code { code: 'b', system: 'a', display: '', version: '' } })
define D580: (Code { system: 'A', display: '', version: null } !~ Code { code: null, system: 'A', display: '', version: 'A' })
define D581: ({ Code { code: 'A', system: 'b', display: null, version: null } } != { Code '' from "S" display 'b' })
define D582: (Concept { codes: { Code { code: 'b', system: '', display: '', version: 'a' } }, display: 'A' } = (null as Concept))
define D583: ({ Code { code: 'a', system: 'b', display: 'A', version: 'A' } } != { Code { code: 'b', system: 'A', display: '', version: 'a' } })
define D584: (Concept { codes: { Code { code: null, system: 'a', display: null, version: 'A' } }, display: 'b' } = Concept { codes: { Code { code: null, system: 'A', display: 'A', version: 'b' } }, display: null })
define D585: (Code { code: 'A', system: 'A', display: 'a', version: 'a' } ~ Code { code: 'A', system: null, display: 'a', version: 'a' })
define D586: ((Code { code: null, system: 'b', display: 'a', version: 'a' }).display is null)
define D587: (Code { system: null, display: 'a', version: 'a' } != Code { system: 'a', version: 'b' })
define D588: (("Z").display is null)
define D589: ((Code { code: 'A', system: 'b', display: '', version: null }).system is null)
define D590: (Code { system: '', display: null, version: 'b' } != Code { code: '', system: '', display: 'A', version: null })
define D591: (Code { code: 'b', system: 'a', display: 'A', version: 'A' } !~ Code { system: 'a', display: 'A', version: 'A' })
define D592: ((Concept { codes: { Code { system: null, display: 'A', version: 'a' } }, display: null }).display is null)
define D593: ({ Code { code: 'a', system: 'a', version: null } } != { Code { code: 'a', system: 'a', version: '' } })
define D594: (Code { code: '', system: null, display: null, version: '' }).display
define D595: (Concept { codes: { Code { code: null, system: 'a', display: 'A', version: '' } }, display: '' } = Concept { codes: { Code { code: null, display: 'a', version: null } }, display: null })
define D596: (Code { code: 'a', system: 'A', display: 'A', version: 'a' }).system
define D597: (Code { code: 'a', system: 'A', display: '' } = Code { code: 'b', system: 'a', display: 'A', version: 'A' })
define D598: (Code { code: null, system: null, display: null, version: 'b' } = Code { code: '', system: '', version: 'A' })
define D599: ({ Code { code: 'A', system: null, display: '', version: 'a' } } = { Code { code: 'b', system: null, version: '' } })
