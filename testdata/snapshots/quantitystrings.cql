library Differential version '1.0'
// Written by hand: the strings of docs/plans/010-to-quantity.md, each read as
// ToQuantity, ConvertsToQuantity and convert to Quantity, and the Integer, Long
// and Decimal overloads.
// CQL Reference (Appendix B), ToQuantity: "For the String overload, the operator
// accepts strings using the following format: (+|-)?#0(.0#)?('<unit>')?", the
// unit "a string literal specifying a valid, case-sensitive UCUM unit of measure
// or calendar duration keyword", "Spaces are allowed between the quantity value
// and the unit designator", and "If the input string is not formatted correctly,
// or cannot be interpreted as a valid Quantity value, the result is null".
define D0: ToQuantity('5 \'mg\'')
define D1: ConvertsToQuantity('5 \'mg\'')
define D2: convert '5 \'mg\'' to Quantity
define D3: ToQuantity('5')
define D4: ConvertsToQuantity('5')
define D5: convert '5' to Quantity
define D6: ToQuantity('5\'mg\'')
define D7: ConvertsToQuantity('5\'mg\'')
define D8: convert '5\'mg\'' to Quantity
define D9: ToQuantity('-5 \'mg\'')
define D10: ConvertsToQuantity('-5 \'mg\'')
define D11: convert '-5 \'mg\'' to Quantity
define D12: ToQuantity('-5')
define D13: ConvertsToQuantity('-5')
define D14: convert '-5' to Quantity
define D15: ToQuantity('-5\'mg\'')
define D16: ConvertsToQuantity('-5\'mg\'')
define D17: convert '-5\'mg\'' to Quantity
define D18: ToQuantity('+5 \'mg\'')
define D19: ConvertsToQuantity('+5 \'mg\'')
define D20: convert '+5 \'mg\'' to Quantity
define D21: ToQuantity('+5')
define D22: ConvertsToQuantity('+5')
define D23: convert '+5' to Quantity
define D24: ToQuantity('+5\'mg\'')
define D25: ConvertsToQuantity('+5\'mg\'')
define D26: convert '+5\'mg\'' to Quantity
define D27: ToQuantity('0 \'mg\'')
define D28: ConvertsToQuantity('0 \'mg\'')
define D29: convert '0 \'mg\'' to Quantity
define D30: ToQuantity('0')
define D31: ConvertsToQuantity('0')
define D32: convert '0' to Quantity
define D33: ToQuantity('0\'mg\'')
define D34: ConvertsToQuantity('0\'mg\'')
define D35: convert '0\'mg\'' to Quantity
define D36: ToQuantity('-0 \'mg\'')
define D37: ConvertsToQuantity('-0 \'mg\'')
define D38: convert '-0 \'mg\'' to Quantity
define D39: ToQuantity('-0')
define D40: ConvertsToQuantity('-0')
define D41: convert '-0' to Quantity
define D42: ToQuantity('-0\'mg\'')
define D43: ConvertsToQuantity('-0\'mg\'')
define D44: convert '-0\'mg\'' to Quantity
define D45: ToQuantity('+0 \'mg\'')
define D46: ConvertsToQuantity('+0 \'mg\'')
define D47: convert '+0 \'mg\'' to Quantity
define D48: ToQuantity('+0')
define D49: ConvertsToQuantity('+0')
define D50: convert '+0' to Quantity
define D51: ToQuantity('+0\'mg\'')
define D52: ConvertsToQuantity('+0\'mg\'')
define D53: convert '+0\'mg\'' to Quantity
define D54: ToQuantity('5.5 \'mg\'')
define D55: ConvertsToQuantity('5.5 \'mg\'')
define D56: convert '5.5 \'mg\'' to Quantity
define D57: ToQuantity('5.5')
define D58: ConvertsToQuantity('5.5')
define D59: convert '5.5' to Quantity
define D60: ToQuantity('5.5\'mg\'')
define D61: ConvertsToQuantity('5.5\'mg\'')
define D62: convert '5.5\'mg\'' to Quantity
define D63: ToQuantity('5.50 \'mg\'')
define D64: ConvertsToQuantity('5.50 \'mg\'')
define D65: convert '5.50 \'mg\'' to Quantity
define D66: ToQuantity('5.50')
define D67: ConvertsToQuantity('5.50')
define D68: convert '5.50' to Quantity
define D69: ToQuantity('5.50\'mg\'')
define D70: ConvertsToQuantity('5.50\'mg\'')
define D71: convert '5.50\'mg\'' to Quantity
define D72: ToQuantity('-5.5 \'mg\'')
define D73: ConvertsToQuantity('-5.5 \'mg\'')
define D74: convert '-5.5 \'mg\'' to Quantity
define D75: ToQuantity('-5.5')
define D76: ConvertsToQuantity('-5.5')
define D77: convert '-5.5' to Quantity
define D78: ToQuantity('-5.5\'mg\'')
define D79: ConvertsToQuantity('-5.5\'mg\'')
define D80: convert '-5.5\'mg\'' to Quantity
define D81: ToQuantity('00005 \'mg\'')
define D82: ConvertsToQuantity('00005 \'mg\'')
define D83: convert '00005 \'mg\'' to Quantity
define D84: ToQuantity('00005')
define D85: ConvertsToQuantity('00005')
define D86: convert '00005' to Quantity
define D87: ToQuantity('00005\'mg\'')
define D88: ConvertsToQuantity('00005\'mg\'')
define D89: convert '00005\'mg\'' to Quantity
define D90: ToQuantity('005.500 \'mg\'')
define D91: ConvertsToQuantity('005.500 \'mg\'')
define D92: convert '005.500 \'mg\'' to Quantity
define D93: ToQuantity('005.500')
define D94: ConvertsToQuantity('005.500')
define D95: convert '005.500' to Quantity
define D96: ToQuantity('005.500\'mg\'')
define D97: ConvertsToQuantity('005.500\'mg\'')
define D98: convert '005.500\'mg\'' to Quantity
define D99: ToQuantity('5. \'mg\'')
define D100: ConvertsToQuantity('5. \'mg\'')
define D101: convert '5. \'mg\'' to Quantity
define D102: ToQuantity('5.')
define D103: ConvertsToQuantity('5.')
define D104: convert '5.' to Quantity
define D105: ToQuantity('5.\'mg\'')
define D106: ConvertsToQuantity('5.\'mg\'')
define D107: convert '5.\'mg\'' to Quantity
define D108: ToQuantity('-5. \'mg\'')
define D109: ConvertsToQuantity('-5. \'mg\'')
define D110: convert '-5. \'mg\'' to Quantity
define D111: ToQuantity('-5.')
define D112: ConvertsToQuantity('-5.')
define D113: convert '-5.' to Quantity
define D114: ToQuantity('-5.\'mg\'')
define D115: ConvertsToQuantity('-5.\'mg\'')
define D116: convert '-5.\'mg\'' to Quantity
define D117: ToQuantity('1e3 \'mg\'')
define D118: ConvertsToQuantity('1e3 \'mg\'')
define D119: convert '1e3 \'mg\'' to Quantity
define D120: ToQuantity('1e3')
define D121: ConvertsToQuantity('1e3')
define D122: convert '1e3' to Quantity
define D123: ToQuantity('1e3\'mg\'')
define D124: ConvertsToQuantity('1e3\'mg\'')
define D125: convert '1e3\'mg\'' to Quantity
define D126: ToQuantity('1E3 \'mg\'')
define D127: ConvertsToQuantity('1E3 \'mg\'')
define D128: convert '1E3 \'mg\'' to Quantity
define D129: ToQuantity('1E3')
define D130: ConvertsToQuantity('1E3')
define D131: convert '1E3' to Quantity
define D132: ToQuantity('1E3\'mg\'')
define D133: ConvertsToQuantity('1E3\'mg\'')
define D134: convert '1E3\'mg\'' to Quantity
define D135: ToQuantity('1e-3 \'mg\'')
define D136: ConvertsToQuantity('1e-3 \'mg\'')
define D137: convert '1e-3 \'mg\'' to Quantity
define D138: ToQuantity('1e-3')
define D139: ConvertsToQuantity('1e-3')
define D140: convert '1e-3' to Quantity
define D141: ToQuantity('1e-3\'mg\'')
define D142: ConvertsToQuantity('1e-3\'mg\'')
define D143: convert '1e-3\'mg\'' to Quantity
define D144: ToQuantity('1e+3 \'mg\'')
define D145: ConvertsToQuantity('1e+3 \'mg\'')
define D146: convert '1e+3 \'mg\'' to Quantity
define D147: ToQuantity('1e+3')
define D148: ConvertsToQuantity('1e+3')
define D149: convert '1e+3' to Quantity
define D150: ToQuantity('1e+3\'mg\'')
define D151: ConvertsToQuantity('1e+3\'mg\'')
define D152: convert '1e+3\'mg\'' to Quantity
define D153: ToQuantity('-1e3 \'mg\'')
define D154: ConvertsToQuantity('-1e3 \'mg\'')
define D155: convert '-1e3 \'mg\'' to Quantity
define D156: ToQuantity('-1e3')
define D157: ConvertsToQuantity('-1e3')
define D158: convert '-1e3' to Quantity
define D159: ToQuantity('-1e3\'mg\'')
define D160: ConvertsToQuantity('-1e3\'mg\'')
define D161: convert '-1e3\'mg\'' to Quantity
define D162: ToQuantity('1.5e2 \'mg\'')
define D163: ConvertsToQuantity('1.5e2 \'mg\'')
define D164: convert '1.5e2 \'mg\'' to Quantity
define D165: ToQuantity('1.5e2')
define D166: ConvertsToQuantity('1.5e2')
define D167: convert '1.5e2' to Quantity
define D168: ToQuantity('1.5e2\'mg\'')
define D169: ConvertsToQuantity('1.5e2\'mg\'')
define D170: convert '1.5e2\'mg\'' to Quantity
define D171: ToQuantity('0e5 \'mg\'')
define D172: ConvertsToQuantity('0e5 \'mg\'')
define D173: convert '0e5 \'mg\'' to Quantity
define D174: ToQuantity('0e5')
define D175: ConvertsToQuantity('0e5')
define D176: convert '0e5' to Quantity
define D177: ToQuantity('0e5\'mg\'')
define D178: ConvertsToQuantity('0e5\'mg\'')
define D179: convert '0e5\'mg\'' to Quantity
define D180: ToQuantity('1e-10 \'mg\'')
define D181: ConvertsToQuantity('1e-10 \'mg\'')
define D182: convert '1e-10 \'mg\'' to Quantity
define D183: ToQuantity('1e-10')
define D184: ConvertsToQuantity('1e-10')
define D185: convert '1e-10' to Quantity
define D186: ToQuantity('1e-10\'mg\'')
define D187: ConvertsToQuantity('1e-10\'mg\'')
define D188: convert '1e-10\'mg\'' to Quantity
define D189: ToQuantity('1.000000005 \'mg\'')
define D190: ConvertsToQuantity('1.000000005 \'mg\'')
define D191: convert '1.000000005 \'mg\'' to Quantity
define D192: ToQuantity('1.000000005')
define D193: ConvertsToQuantity('1.000000005')
define D194: convert '1.000000005' to Quantity
define D195: ToQuantity('1.000000005\'mg\'')
define D196: ConvertsToQuantity('1.000000005\'mg\'')
define D197: convert '1.000000005\'mg\'' to Quantity
define D198: ToQuantity('0.000000001 \'mg\'')
define D199: ConvertsToQuantity('0.000000001 \'mg\'')
define D200: convert '0.000000001 \'mg\'' to Quantity
define D201: ToQuantity('0.000000001')
define D202: ConvertsToQuantity('0.000000001')
define D203: convert '0.000000001' to Quantity
define D204: ToQuantity('0.000000001\'mg\'')
define D205: ConvertsToQuantity('0.000000001\'mg\'')
define D206: convert '0.000000001\'mg\'' to Quantity
define D207: ToQuantity('-0.000000001 \'mg\'')
define D208: ConvertsToQuantity('-0.000000001 \'mg\'')
define D209: convert '-0.000000001 \'mg\'' to Quantity
define D210: ToQuantity('-0.000000001')
define D211: ConvertsToQuantity('-0.000000001')
define D212: convert '-0.000000001' to Quantity
define D213: ToQuantity('-0.000000001\'mg\'')
define D214: ConvertsToQuantity('-0.000000001\'mg\'')
define D215: convert '-0.000000001\'mg\'' to Quantity
define D216: ToQuantity('-1.00000001 \'mg\'')
define D217: ConvertsToQuantity('-1.00000001 \'mg\'')
define D218: convert '-1.00000001 \'mg\'' to Quantity
define D219: ToQuantity('-1.00000001')
define D220: ConvertsToQuantity('-1.00000001')
define D221: convert '-1.00000001' to Quantity
define D222: ToQuantity('-1.00000001\'mg\'')
define D223: ConvertsToQuantity('-1.00000001\'mg\'')
define D224: convert '-1.00000001\'mg\'' to Quantity
define D225: ToQuantity('1.123456789 \'mg\'')
define D226: ConvertsToQuantity('1.123456789 \'mg\'')
define D227: convert '1.123456789 \'mg\'' to Quantity
define D228: ToQuantity('1.123456789')
define D229: ConvertsToQuantity('1.123456789')
define D230: convert '1.123456789' to Quantity
define D231: ToQuantity('1.123456789\'mg\'')
define D232: ConvertsToQuantity('1.123456789\'mg\'')
define D233: convert '1.123456789\'mg\'' to Quantity
define D234: ToQuantity('0.000000000 \'mg\'')
define D235: ConvertsToQuantity('0.000000000 \'mg\'')
define D236: convert '0.000000000 \'mg\'' to Quantity
define D237: ToQuantity('0.000000000')
define D238: ConvertsToQuantity('0.000000000')
define D239: convert '0.000000000' to Quantity
define D240: ToQuantity('0.000000000\'mg\'')
define D241: ConvertsToQuantity('0.000000000\'mg\'')
define D242: convert '0.000000000\'mg\'' to Quantity
define D243: ToQuantity('12345678901234567890.12345678 \'mg\'')
define D244: ConvertsToQuantity('12345678901234567890.12345678 \'mg\'')
define D245: convert '12345678901234567890.12345678 \'mg\'' to Quantity
define D246: ToQuantity('12345678901234567890.12345678')
define D247: ConvertsToQuantity('12345678901234567890.12345678')
define D248: convert '12345678901234567890.12345678' to Quantity
define D249: ToQuantity('12345678901234567890.12345678\'mg\'')
define D250: ConvertsToQuantity('12345678901234567890.12345678\'mg\'')
define D251: convert '12345678901234567890.12345678\'mg\'' to Quantity
define D252: ToQuantity('99999999999999999999 \'mg\'')
define D253: ConvertsToQuantity('99999999999999999999 \'mg\'')
define D254: convert '99999999999999999999 \'mg\'' to Quantity
define D255: ToQuantity('99999999999999999999')
define D256: ConvertsToQuantity('99999999999999999999')
define D257: convert '99999999999999999999' to Quantity
define D258: ToQuantity('99999999999999999999\'mg\'')
define D259: ConvertsToQuantity('99999999999999999999\'mg\'')
define D260: convert '99999999999999999999\'mg\'' to Quantity
define D261: ToQuantity('100000000000000000000 \'mg\'')
define D262: ConvertsToQuantity('100000000000000000000 \'mg\'')
define D263: convert '100000000000000000000 \'mg\'' to Quantity
define D264: ToQuantity('100000000000000000000')
define D265: ConvertsToQuantity('100000000000000000000')
define D266: convert '100000000000000000000' to Quantity
define D267: ToQuantity('100000000000000000000\'mg\'')
define D268: ConvertsToQuantity('100000000000000000000\'mg\'')
define D269: convert '100000000000000000000\'mg\'' to Quantity
define D270: ToQuantity('-99999999999999999999.99999999 \'mg\'')
define D271: ConvertsToQuantity('-99999999999999999999.99999999 \'mg\'')
define D272: convert '-99999999999999999999.99999999 \'mg\'' to Quantity
define D273: ToQuantity('-99999999999999999999.99999999')
define D274: ConvertsToQuantity('-99999999999999999999.99999999')
define D275: convert '-99999999999999999999.99999999' to Quantity
define D276: ToQuantity('-99999999999999999999.99999999\'mg\'')
define D277: ConvertsToQuantity('-99999999999999999999.99999999\'mg\'')
define D278: convert '-99999999999999999999.99999999\'mg\'' to Quantity
define D279: ToQuantity('99999999999999999999.999999991 \'mg\'')
define D280: ConvertsToQuantity('99999999999999999999.999999991 \'mg\'')
define D281: convert '99999999999999999999.999999991 \'mg\'' to Quantity
define D282: ToQuantity('99999999999999999999.999999991')
define D283: ConvertsToQuantity('99999999999999999999.999999991')
define D284: convert '99999999999999999999.999999991' to Quantity
define D285: ToQuantity('99999999999999999999.999999991\'mg\'')
define D286: ConvertsToQuantity('99999999999999999999.999999991\'mg\'')
define D287: convert '99999999999999999999.999999991\'mg\'' to Quantity
define D288: ToQuantity('-100000000000000000000 \'mg\'')
define D289: ConvertsToQuantity('-100000000000000000000 \'mg\'')
define D290: convert '-100000000000000000000 \'mg\'' to Quantity
define D291: ToQuantity('-100000000000000000000')
define D292: ConvertsToQuantity('-100000000000000000000')
define D293: convert '-100000000000000000000' to Quantity
define D294: ToQuantity('-100000000000000000000\'mg\'')
define D295: ConvertsToQuantity('-100000000000000000000\'mg\'')
define D296: convert '-100000000000000000000\'mg\'' to Quantity
define D297: ToQuantity('1e10001 \'mg\'')
define D298: ConvertsToQuantity('1e10001 \'mg\'')
define D299: convert '1e10001 \'mg\'' to Quantity
define D300: ToQuantity('1e10001')
define D301: ConvertsToQuantity('1e10001')
define D302: convert '1e10001' to Quantity
define D303: ToQuantity('1e10001\'mg\'')
define D304: ConvertsToQuantity('1e10001\'mg\'')
define D305: convert '1e10001\'mg\'' to Quantity
define D306: ToQuantity('1e-10001 \'mg\'')
define D307: ConvertsToQuantity('1e-10001 \'mg\'')
define D308: convert '1e-10001 \'mg\'' to Quantity
define D309: ToQuantity('1e-10001')
define D310: ConvertsToQuantity('1e-10001')
define D311: convert '1e-10001' to Quantity
define D312: ToQuantity('1e-10001\'mg\'')
define D313: ConvertsToQuantity('1e-10001\'mg\'')
define D314: convert '1e-10001\'mg\'' to Quantity
define D315: ToQuantity('1e19 \'mg\'')
define D316: ConvertsToQuantity('1e19 \'mg\'')
define D317: convert '1e19 \'mg\'' to Quantity
define D318: ToQuantity('1e19')
define D319: ConvertsToQuantity('1e19')
define D320: convert '1e19' to Quantity
define D321: ToQuantity('1e19\'mg\'')
define D322: ConvertsToQuantity('1e19\'mg\'')
define D323: convert '1e19\'mg\'' to Quantity
define D324: ToQuantity('1e20 \'mg\'')
define D325: ConvertsToQuantity('1e20 \'mg\'')
define D326: convert '1e20 \'mg\'' to Quantity
define D327: ToQuantity('1e20')
define D328: ConvertsToQuantity('1e20')
define D329: convert '1e20' to Quantity
define D330: ToQuantity('1e20\'mg\'')
define D331: ConvertsToQuantity('1e20\'mg\'')
define D332: convert '1e20\'mg\'' to Quantity
define D333: ToQuantity('.5 \'mg\'')
define D334: ConvertsToQuantity('.5 \'mg\'')
define D335: convert '.5 \'mg\'' to Quantity
define D336: ToQuantity('.5')
define D337: ConvertsToQuantity('.5')
define D338: convert '.5' to Quantity
define D339: ToQuantity('.5\'mg\'')
define D340: ConvertsToQuantity('.5\'mg\'')
define D341: convert '.5\'mg\'' to Quantity
define D342: ToQuantity('+.5 \'mg\'')
define D343: ConvertsToQuantity('+.5 \'mg\'')
define D344: convert '+.5 \'mg\'' to Quantity
define D345: ToQuantity('+.5')
define D346: ConvertsToQuantity('+.5')
define D347: convert '+.5' to Quantity
define D348: ToQuantity('+.5\'mg\'')
define D349: ConvertsToQuantity('+.5\'mg\'')
define D350: convert '+.5\'mg\'' to Quantity
define D351: ToQuantity('-.5 \'mg\'')
define D352: ConvertsToQuantity('-.5 \'mg\'')
define D353: convert '-.5 \'mg\'' to Quantity
define D354: ToQuantity('-.5')
define D355: ConvertsToQuantity('-.5')
define D356: convert '-.5' to Quantity
define D357: ToQuantity('-.5\'mg\'')
define D358: ConvertsToQuantity('-.5\'mg\'')
define D359: convert '-.5\'mg\'' to Quantity
define D360: ToQuantity('−5 \'mg\'')
define D361: ConvertsToQuantity('−5 \'mg\'')
define D362: convert '−5 \'mg\'' to Quantity
define D363: ToQuantity('−5')
define D364: ConvertsToQuantity('−5')
define D365: convert '−5' to Quantity
define D366: ToQuantity('−5\'mg\'')
define D367: ConvertsToQuantity('−5\'mg\'')
define D368: convert '−5\'mg\'' to Quantity
define D369: ToQuantity('.-5 \'mg\'')
define D370: ConvertsToQuantity('.-5 \'mg\'')
define D371: convert '.-5 \'mg\'' to Quantity
define D372: ToQuantity('.-5')
define D373: ConvertsToQuantity('.-5')
define D374: convert '.-5' to Quantity
define D375: ToQuantity('.-5\'mg\'')
define D376: ConvertsToQuantity('.-5\'mg\'')
define D377: convert '.-5\'mg\'' to Quantity
define D378: ToQuantity('5..5 \'mg\'')
define D379: ConvertsToQuantity('5..5 \'mg\'')
define D380: convert '5..5 \'mg\'' to Quantity
define D381: ToQuantity('5..5')
define D382: ConvertsToQuantity('5..5')
define D383: convert '5..5' to Quantity
define D384: ToQuantity('5..5\'mg\'')
define D385: ConvertsToQuantity('5..5\'mg\'')
define D386: convert '5..5\'mg\'' to Quantity
define D387: ToQuantity('5.5.5 \'mg\'')
define D388: ConvertsToQuantity('5.5.5 \'mg\'')
define D389: convert '5.5.5 \'mg\'' to Quantity
define D390: ToQuantity('5.5.5')
define D391: ConvertsToQuantity('5.5.5')
define D392: convert '5.5.5' to Quantity
define D393: ToQuantity('5.5.5\'mg\'')
define D394: ConvertsToQuantity('5.5.5\'mg\'')
define D395: convert '5.5.5\'mg\'' to Quantity
define D396: ToQuantity('--5 \'mg\'')
define D397: ConvertsToQuantity('--5 \'mg\'')
define D398: convert '--5 \'mg\'' to Quantity
define D399: ToQuantity('--5')
define D400: ConvertsToQuantity('--5')
define D401: convert '--5' to Quantity
define D402: ToQuantity('--5\'mg\'')
define D403: ConvertsToQuantity('--5\'mg\'')
define D404: convert '--5\'mg\'' to Quantity
define D405: ToQuantity('+-5 \'mg\'')
define D406: ConvertsToQuantity('+-5 \'mg\'')
define D407: convert '+-5 \'mg\'' to Quantity
define D408: ToQuantity('+-5')
define D409: ConvertsToQuantity('+-5')
define D410: convert '+-5' to Quantity
define D411: ToQuantity('+-5\'mg\'')
define D412: ConvertsToQuantity('+-5\'mg\'')
define D413: convert '+-5\'mg\'' to Quantity
define D414: ToQuantity('e3 \'mg\'')
define D415: ConvertsToQuantity('e3 \'mg\'')
define D416: convert 'e3 \'mg\'' to Quantity
define D417: ToQuantity('e3')
define D418: ConvertsToQuantity('e3')
define D419: convert 'e3' to Quantity
define D420: ToQuantity('e3\'mg\'')
define D421: ConvertsToQuantity('e3\'mg\'')
define D422: convert 'e3\'mg\'' to Quantity
define D423: ToQuantity('1e \'mg\'')
define D424: ConvertsToQuantity('1e \'mg\'')
define D425: convert '1e \'mg\'' to Quantity
define D426: ToQuantity('1e')
define D427: ConvertsToQuantity('1e')
define D428: convert '1e' to Quantity
define D429: ToQuantity('1e\'mg\'')
define D430: ConvertsToQuantity('1e\'mg\'')
define D431: convert '1e\'mg\'' to Quantity
define D432: ToQuantity('1e+ \'mg\'')
define D433: ConvertsToQuantity('1e+ \'mg\'')
define D434: convert '1e+ \'mg\'' to Quantity
define D435: ToQuantity('1e+')
define D436: ConvertsToQuantity('1e+')
define D437: convert '1e+' to Quantity
define D438: ToQuantity('1e+\'mg\'')
define D439: ConvertsToQuantity('1e+\'mg\'')
define D440: convert '1e+\'mg\'' to Quantity
define D441: ToQuantity('0x10 \'mg\'')
define D442: ConvertsToQuantity('0x10 \'mg\'')
define D443: convert '0x10 \'mg\'' to Quantity
define D444: ToQuantity('0x10')
define D445: ConvertsToQuantity('0x10')
define D446: convert '0x10' to Quantity
define D447: ToQuantity('0x10\'mg\'')
define D448: ConvertsToQuantity('0x10\'mg\'')
define D449: convert '0x10\'mg\'' to Quantity
define D450: ToQuantity('٥ \'mg\'')
define D451: ConvertsToQuantity('٥ \'mg\'')
define D452: convert '٥ \'mg\'' to Quantity
define D453: ToQuantity('٥')
define D454: ConvertsToQuantity('٥')
define D455: convert '٥' to Quantity
define D456: ToQuantity('٥\'mg\'')
define D457: ConvertsToQuantity('٥\'mg\'')
define D458: convert '٥\'mg\'' to Quantity
define D459: ToQuantity('5,5 \'mg\'')
define D460: ConvertsToQuantity('5,5 \'mg\'')
define D461: convert '5,5 \'mg\'' to Quantity
define D462: ToQuantity('5,5')
define D463: ConvertsToQuantity('5,5')
define D464: convert '5,5' to Quantity
define D465: ToQuantity('5,5\'mg\'')
define D466: ConvertsToQuantity('5,5\'mg\'')
define D467: convert '5,5\'mg\'' to Quantity
define D468: ToQuantity('1_000 \'mg\'')
define D469: ConvertsToQuantity('1_000 \'mg\'')
define D470: convert '1_000 \'mg\'' to Quantity
define D471: ToQuantity('1_000')
define D472: ConvertsToQuantity('1_000')
define D473: convert '1_000' to Quantity
define D474: ToQuantity('1_000\'mg\'')
define D475: ConvertsToQuantity('1_000\'mg\'')
define D476: convert '1_000\'mg\'' to Quantity
define D477: ToQuantity('Infinity \'mg\'')
define D478: ConvertsToQuantity('Infinity \'mg\'')
define D479: convert 'Infinity \'mg\'' to Quantity
define D480: ToQuantity('Infinity')
define D481: ConvertsToQuantity('Infinity')
define D482: convert 'Infinity' to Quantity
define D483: ToQuantity('Infinity\'mg\'')
define D484: ConvertsToQuantity('Infinity\'mg\'')
define D485: convert 'Infinity\'mg\'' to Quantity
define D486: ToQuantity('NaN \'mg\'')
define D487: ConvertsToQuantity('NaN \'mg\'')
define D488: convert 'NaN \'mg\'' to Quantity
define D489: ToQuantity('NaN')
define D490: ConvertsToQuantity('NaN')
define D491: convert 'NaN' to Quantity
define D492: ToQuantity('NaN\'mg\'')
define D493: ConvertsToQuantity('NaN\'mg\'')
define D494: convert 'NaN\'mg\'' to Quantity
define D495: ToQuantity('- 5 \'mg\'')
define D496: ConvertsToQuantity('- 5 \'mg\'')
define D497: convert '- 5 \'mg\'' to Quantity
define D498: ToQuantity('- 5')
define D499: ConvertsToQuantity('- 5')
define D500: convert '- 5' to Quantity
define D501: ToQuantity('- 5\'mg\'')
define D502: ConvertsToQuantity('- 5\'mg\'')
define D503: convert '- 5\'mg\'' to Quantity
define D504: ToQuantity('5- \'mg\'')
define D505: ConvertsToQuantity('5- \'mg\'')
define D506: convert '5- \'mg\'' to Quantity
define D507: ToQuantity('5-')
define D508: ConvertsToQuantity('5-')
define D509: convert '5-' to Quantity
define D510: ToQuantity('5-\'mg\'')
define D511: ConvertsToQuantity('5-\'mg\'')
define D512: convert '5-\'mg\'' to Quantity
define D513: ToQuantity('5 \'foo\'')
define D514: ConvertsToQuantity('5 \'foo\'')
define D515: convert '5 \'foo\'' to Quantity
define D516: ToQuantity('5 \'MG\'')
define D517: ConvertsToQuantity('5 \'MG\'')
define D518: convert '5 \'MG\'' to Quantity
define D519: ToQuantity('5 \'Mg\'')
define D520: ConvertsToQuantity('5 \'Mg\'')
define D521: convert '5 \'Mg\'' to Quantity
define D522: ToQuantity('5 \'m g\'')
define D523: ConvertsToQuantity('5 \'m g\'')
define D524: convert '5 \'m g\'' to Quantity
define D525: ToQuantity('5 \'{tablet}\'')
define D526: ConvertsToQuantity('5 \'{tablet}\'')
define D527: convert '5 \'{tablet}\'' to Quantity
define D528: ToQuantity('5 \'mg/dL\'')
define D529: ConvertsToQuantity('5 \'mg/dL\'')
define D530: convert '5 \'mg/dL\'' to Quantity
define D531: ToQuantity('5 \'[lb_av]\'')
define D532: ConvertsToQuantity('5 \'[lb_av]\'')
define D533: convert '5 \'[lb_av]\'' to Quantity
define D534: ToQuantity('5 \'1\'')
define D535: ConvertsToQuantity('5 \'1\'')
define D536: convert '5 \'1\'' to Quantity
define D537: ToQuantity('5 \'d\'')
define D538: ConvertsToQuantity('5 \'d\'')
define D539: convert '5 \'d\'' to Quantity
define D540: ToQuantity('5 \'mo\'')
define D541: ConvertsToQuantity('5 \'mo\'')
define D542: convert '5 \'mo\'' to Quantity
define D543: ToQuantity('5 \'wk\'')
define D544: ConvertsToQuantity('5 \'wk\'')
define D545: convert '5 \'wk\'' to Quantity
define D546: ToQuantity('5 \'a\'')
define D547: ConvertsToQuantity('5 \'a\'')
define D548: convert '5 \'a\'' to Quantity
define D549: ToQuantity('5 \'%\'')
define D550: ConvertsToQuantity('5 \'%\'')
define D551: convert '5 \'%\'' to Quantity
define D552: ToQuantity('5 \'10*3/uL\'')
define D553: ConvertsToQuantity('5 \'10*3/uL\'')
define D554: convert '5 \'10*3/uL\'' to Quantity
define D555: ToQuantity('5 \'\'')
define D556: ConvertsToQuantity('5 \'\'')
define D557: convert '5 \'\'' to Quantity
define D558: ToQuantity('5 \'kg.m/s2\'')
define D559: ConvertsToQuantity('5 \'kg.m/s2\'')
define D560: convert '5 \'kg.m/s2\'' to Quantity
define D561: ToQuantity('5 \'cm[H2O]\'')
define D562: ConvertsToQuantity('5 \'cm[H2O]\'')
define D563: convert '5 \'cm[H2O]\'' to Quantity
define D564: ToQuantity('5 \'Cel\'')
define D565: ConvertsToQuantity('5 \'Cel\'')
define D566: convert '5 \'Cel\'' to Quantity
define D567: ToQuantity('5 \'[degF]\'')
define D568: ConvertsToQuantity('5 \'[degF]\'')
define D569: convert '5 \'[degF]\'' to Quantity
define D570: ToQuantity('5 \'ug/(kg.h)\'')
define D571: ConvertsToQuantity('5 \'ug/(kg.h)\'')
define D572: convert '5 \'ug/(kg.h)\'' to Quantity
define D573: ToQuantity('5 \'mmol/L\'')
define D574: ConvertsToQuantity('5 \'mmol/L\'')
define D575: convert '5 \'mmol/L\'' to Quantity
define D576: ToQuantity('5 \'/min\'')
define D577: ConvertsToQuantity('5 \'/min\'')
define D578: convert '5 \'/min\'' to Quantity
define D579: ToQuantity('5 \'g/\'')
define D580: ConvertsToQuantity('5 \'g/\'')
define D581: convert '5 \'g/\'' to Quantity
define D582: ToQuantity('5 \'mgx\'')
define D583: ConvertsToQuantity('5 \'mgx\'')
define D584: convert '5 \'mgx\'' to Quantity
define D585: ToQuantity('5 \'DAYS\'')
define D586: ConvertsToQuantity('5 \'DAYS\'')
define D587: convert '5 \'DAYS\'' to Quantity
define D588: ToQuantity('5 \'Day\'')
define D589: ConvertsToQuantity('5 \'Day\'')
define D590: convert '5 \'Day\'' to Quantity
define D591: ToQuantity('5 \'DAY\'')
define D592: ConvertsToQuantity('5 \'DAY\'')
define D593: convert '5 \'DAY\'' to Quantity
define D594: ToQuantity('5 \'y\'')
define D595: ConvertsToQuantity('5 \'y\'')
define D596: convert '5 \'y\'' to Quantity
define D597: ToQuantity('5 \'yr\'')
define D598: ConvertsToQuantity('5 \'yr\'')
define D599: convert '5 \'yr\'' to Quantity
define D600: ToQuantity('5 \'year\'')
define D601: ConvertsToQuantity('5 \'year\'')
define D602: convert '5 \'year\'' to Quantity
define D603: ToQuantity('5 \'years\'')
define D604: ConvertsToQuantity('5 \'years\'')
define D605: convert '5 \'years\'' to Quantity
define D606: ToQuantity('5 \'month\'')
define D607: ConvertsToQuantity('5 \'month\'')
define D608: convert '5 \'month\'' to Quantity
define D609: ToQuantity('5 \'months\'')
define D610: ConvertsToQuantity('5 \'months\'')
define D611: convert '5 \'months\'' to Quantity
define D612: ToQuantity('5 \'week\'')
define D613: ConvertsToQuantity('5 \'week\'')
define D614: convert '5 \'week\'' to Quantity
define D615: ToQuantity('5 \'weeks\'')
define D616: ConvertsToQuantity('5 \'weeks\'')
define D617: convert '5 \'weeks\'' to Quantity
define D618: ToQuantity('5 \'day\'')
define D619: ConvertsToQuantity('5 \'day\'')
define D620: convert '5 \'day\'' to Quantity
define D621: ToQuantity('5 \'days\'')
define D622: ConvertsToQuantity('5 \'days\'')
define D623: convert '5 \'days\'' to Quantity
define D624: ToQuantity('5 \'hour\'')
define D625: ConvertsToQuantity('5 \'hour\'')
define D626: convert '5 \'hour\'' to Quantity
define D627: ToQuantity('5 \'hours\'')
define D628: ConvertsToQuantity('5 \'hours\'')
define D629: convert '5 \'hours\'' to Quantity
define D630: ToQuantity('5 \'minute\'')
define D631: ConvertsToQuantity('5 \'minute\'')
define D632: convert '5 \'minute\'' to Quantity
define D633: ToQuantity('5 \'minutes\'')
define D634: ConvertsToQuantity('5 \'minutes\'')
define D635: convert '5 \'minutes\'' to Quantity
define D636: ToQuantity('5 \'second\'')
define D637: ConvertsToQuantity('5 \'second\'')
define D638: convert '5 \'second\'' to Quantity
define D639: ToQuantity('5 \'seconds\'')
define D640: ConvertsToQuantity('5 \'seconds\'')
define D641: convert '5 \'seconds\'' to Quantity
define D642: ToQuantity('5 \'millisecond\'')
define D643: ConvertsToQuantity('5 \'millisecond\'')
define D644: convert '5 \'millisecond\'' to Quantity
define D645: ToQuantity('5 \'milliseconds\'')
define D646: ConvertsToQuantity('5 \'milliseconds\'')
define D647: convert '5 \'milliseconds\'' to Quantity
define D648: ToQuantity('5 year')
define D649: ConvertsToQuantity('5 year')
define D650: convert '5 year' to Quantity
define D651: ToQuantity('5 years')
define D652: ConvertsToQuantity('5 years')
define D653: convert '5 years' to Quantity
define D654: ToQuantity('5 month')
define D655: ConvertsToQuantity('5 month')
define D656: convert '5 month' to Quantity
define D657: ToQuantity('5 months')
define D658: ConvertsToQuantity('5 months')
define D659: convert '5 months' to Quantity
define D660: ToQuantity('5 week')
define D661: ConvertsToQuantity('5 week')
define D662: convert '5 week' to Quantity
define D663: ToQuantity('5 weeks')
define D664: ConvertsToQuantity('5 weeks')
define D665: convert '5 weeks' to Quantity
define D666: ToQuantity('5 day')
define D667: ConvertsToQuantity('5 day')
define D668: convert '5 day' to Quantity
define D669: ToQuantity('5 days')
define D670: ConvertsToQuantity('5 days')
define D671: convert '5 days' to Quantity
define D672: ToQuantity('5 hour')
define D673: ConvertsToQuantity('5 hour')
define D674: convert '5 hour' to Quantity
define D675: ToQuantity('5 hours')
define D676: ConvertsToQuantity('5 hours')
define D677: convert '5 hours' to Quantity
define D678: ToQuantity('5 minute')
define D679: ConvertsToQuantity('5 minute')
define D680: convert '5 minute' to Quantity
define D681: ToQuantity('5 minutes')
define D682: ConvertsToQuantity('5 minutes')
define D683: convert '5 minutes' to Quantity
define D684: ToQuantity('5 second')
define D685: ConvertsToQuantity('5 second')
define D686: convert '5 second' to Quantity
define D687: ToQuantity('5 seconds')
define D688: ConvertsToQuantity('5 seconds')
define D689: convert '5 seconds' to Quantity
define D690: ToQuantity('5 millisecond')
define D691: ConvertsToQuantity('5 millisecond')
define D692: convert '5 millisecond' to Quantity
define D693: ToQuantity('5 milliseconds')
define D694: ConvertsToQuantity('5 milliseconds')
define D695: convert '5 milliseconds' to Quantity
define D696: ToQuantity('5 mg')
define D697: ConvertsToQuantity('5 mg')
define D698: convert '5 mg' to Quantity
define D699: ToQuantity('5 DAYS')
define D700: ConvertsToQuantity('5 DAYS')
define D701: convert '5 DAYS' to Quantity
define D702: ToQuantity('5 Days')
define D703: ConvertsToQuantity('5 Days')
define D704: convert '5 Days' to Quantity
define D705: ToQuantity('5 5')
define D706: ConvertsToQuantity('5 5')
define D707: convert '5 5' to Quantity
define D708: ToQuantity('5 "mg"')
define D709: ConvertsToQuantity('5 "mg"')
define D710: convert '5 "mg"' to Quantity
define D711: ToQuantity('5 \'')
define D712: ConvertsToQuantity('5 \'')
define D713: convert '5 \'' to Quantity
define D714: ToQuantity('5 \'mg')
define D715: ConvertsToQuantity('5 \'mg')
define D716: convert '5 \'mg' to Quantity
define D717: ToQuantity('5 \'mg\'x')
define D718: ConvertsToQuantity('5 \'mg\'x')
define D719: convert '5 \'mg\'x' to Quantity
define D720: ToQuantity('5 \'mg\' \'g\'')
define D721: ConvertsToQuantity('5 \'mg\' \'g\'')
define D722: convert '5 \'mg\' \'g\'' to Quantity
define D723: ToQuantity('5 \'mg\'\'')
define D724: ConvertsToQuantity('5 \'mg\'\'')
define D725: convert '5 \'mg\'\'' to Quantity
define D726: ToQuantity('5 \'\'mg\'')
define D727: ConvertsToQuantity('5 \'\'mg\'')
define D728: convert '5 \'\'mg\'' to Quantity
define D729: ToQuantity('5 \'m\'g\'')
define D730: ConvertsToQuantity('5 \'m\'g\'')
define D731: convert '5 \'m\'g\'' to Quantity
define D732: ToQuantity('5 \'m\\\'g\'')
define D733: ConvertsToQuantity('5 \'m\\\'g\'')
define D734: convert '5 \'m\\\'g\'' to Quantity
define D735: ToQuantity('5 \'m\\g\'')
define D736: ConvertsToQuantity('5 \'m\\g\'')
define D737: convert '5 \'m\\g\'' to Quantity
define D738: ToQuantity('5 mg\'')
define D739: ConvertsToQuantity('5 mg\'')
define D740: convert '5 mg\'' to Quantity
define D741: ToQuantity('\'mg\' 5')
define D742: ConvertsToQuantity('\'mg\' 5')
define D743: convert '\'mg\' 5' to Quantity
define D744: ToQuantity('mg')
define D745: ConvertsToQuantity('mg')
define D746: convert 'mg' to Quantity
define D747: ToQuantity('abc')
define D748: ConvertsToQuantity('abc')
define D749: convert 'abc' to Quantity
define D750: ToQuantity('')
define D751: ConvertsToQuantity('')
define D752: convert '' to Quantity
define D753: ToQuantity(' ')
define D754: ConvertsToQuantity(' ')
define D755: convert ' ' to Quantity
define D756: ToQuantity('\'mg\'')
define D757: ConvertsToQuantity('\'mg\'')
define D758: convert '\'mg\'' to Quantity
define D759: ToQuantity('5 \'mg\' 5')
define D760: ConvertsToQuantity('5 \'mg\' 5')
define D761: convert '5 \'mg\' 5' to Quantity
define D762: ToQuantity('5 \'mg\'.')
define D763: ConvertsToQuantity('5 \'mg\'.')
define D764: convert '5 \'mg\'.' to Quantity
define D765: ToQuantity('5 days \'mg\'')
define D766: ConvertsToQuantity('5 days \'mg\'')
define D767: convert '5 days \'mg\'' to Quantity
define D768: ToQuantity('5 \'days\' \'mg\'')
define D769: ConvertsToQuantity('5 \'days\' \'mg\'')
define D770: convert '5 \'days\' \'mg\'' to Quantity
define D771: ToQuantity('444 \'cm')
define D772: ConvertsToQuantity('444 \'cm')
define D773: convert '444 \'cm' to Quantity
define D774: ToQuantity('-0.1 \'mg\'')
define D775: ConvertsToQuantity('-0.1 \'mg\'')
define D776: convert '-0.1 \'mg\'' to Quantity
define D777: ToQuantity('5 \'{}\'')
define D778: ConvertsToQuantity('5 \'{}\'')
define D779: convert '5 \'{}\'' to Quantity
define D780: ToQuantity('5 \'mg\'\u0000')
define D781: ConvertsToQuantity('5 \'mg\'\u0000')
define D782: convert '5 \'mg\'\u0000' to Quantity
define D783: ToQuantity(' 5 \'mg\'')
define D784: ConvertsToQuantity(' 5 \'mg\'')
define D785: convert ' 5 \'mg\'' to Quantity
define D786: ToQuantity('5 \'mg\' ')
define D787: ConvertsToQuantity('5 \'mg\' ')
define D788: convert '5 \'mg\' ' to Quantity
define D789: ToQuantity('  5  \'mg\'  ')
define D790: ConvertsToQuantity('  5  \'mg\'  ')
define D791: convert '  5  \'mg\'  ' to Quantity
define D792: ToQuantity('\t5 \'mg\'')
define D793: ConvertsToQuantity('\t5 \'mg\'')
define D794: convert '\t5 \'mg\'' to Quantity
define D795: ToQuantity('5 \'mg\'\t')
define D796: ConvertsToQuantity('5 \'mg\'\t')
define D797: convert '5 \'mg\'\t' to Quantity
define D798: ToQuantity('5\t\'mg\'')
define D799: ConvertsToQuantity('5\t\'mg\'')
define D800: convert '5\t\'mg\'' to Quantity
define D801: ToQuantity('5 \t \'mg\'')
define D802: ConvertsToQuantity('5 \t \'mg\'')
define D803: convert '5 \t \'mg\'' to Quantity
define D804: ToQuantity('\n5 \'mg\'')
define D805: ConvertsToQuantity('\n5 \'mg\'')
define D806: convert '\n5 \'mg\'' to Quantity
define D807: ToQuantity('5 \'mg\'\n')
define D808: ConvertsToQuantity('5 \'mg\'\n')
define D809: convert '5 \'mg\'\n' to Quantity
define D810: ToQuantity('5\n\'mg\'')
define D811: ConvertsToQuantity('5\n\'mg\'')
define D812: convert '5\n\'mg\'' to Quantity
define D813: ToQuantity('5\r\n\'mg\'')
define D814: ConvertsToQuantity('5\r\n\'mg\'')
define D815: convert '5\r\n\'mg\'' to Quantity
define D816: ToQuantity('\r\n5 \'mg\'\r\n')
define D817: ConvertsToQuantity('\r\n5 \'mg\'\r\n')
define D818: convert '\r\n5 \'mg\'\r\n' to Quantity
define D819: ToQuantity('\u00A05 \'mg\'')
define D820: ConvertsToQuantity('\u00A05 \'mg\'')
define D821: convert '\u00A05 \'mg\'' to Quantity
define D822: ToQuantity('5\u00A0\'mg\'')
define D823: ConvertsToQuantity('5\u00A0\'mg\'')
define D824: convert '5\u00A0\'mg\'' to Quantity
define D825: ToQuantity('5 \'mg\'\u00A0')
define D826: ConvertsToQuantity('5 \'mg\'\u00A0')
define D827: convert '5 \'mg\'\u00A0' to Quantity
define D828: ToQuantity(' 5')
define D829: ConvertsToQuantity(' 5')
define D830: convert ' 5' to Quantity
define D831: ToQuantity('5 ')
define D832: ConvertsToQuantity('5 ')
define D833: convert '5 ' to Quantity
define D834: ToQuantity('\t5\t')
define D835: ConvertsToQuantity('\t5\t')
define D836: convert '\t5\t' to Quantity
define D837: ToQuantity('\n5\n')
define D838: ConvertsToQuantity('\n5\n')
define D839: convert '\n5\n' to Quantity
define D840: ToQuantity('5    \'mg\'')
define D841: ConvertsToQuantity('5    \'mg\'')
define D842: convert '5    \'mg\'' to Quantity
define D843: ToQuantity(' 5.5e1 \'mg\' ')
define D844: ConvertsToQuantity(' 5.5e1 \'mg\' ')
define D845: convert ' 5.5e1 \'mg\' ' to Quantity
define D846: ToQuantity(0.1)
define D847: ToQuantity(13)
define D848: ToQuantity(13L)
define D849: ToQuantity(-0.0)
define D850: ToQuantity(5.50)
define D851: ToQuantity(1.000000005)
define D852: ToQuantity(-2147483648)
define D853: ToQuantity(99999999999999999999.99999999)
define D854: ConvertsToQuantity(13)
define D855: ConvertsToQuantity(0.1)
define D856: convert 13 to Quantity
define D857: convert 0.1 to Quantity
define D858: ToQuantity(null as String)
define D859: ToQuantity(null as Integer)
define D860: ToQuantity(null as Decimal)
define D861: ConvertsToQuantity(null as String)
define D862: convert (null as String) to Quantity
define D863: ToQuantity('5 \'mg\'') = 5 'mg'
define D864: ToQuantity('5 \'mg\'') = 5000 'ug'
define D865: ToQuantity('5 \'days\'') = 5 days
define D866: ToQuantity('5') = 5 '1'
define D867: ToQuantity(13) = 13 '1'
define D868: ToQuantity('1.000000005 \'mg\'') = 1.000000005 'mg'
define D869: ToQuantity('-0.000000001 \'mg\'') < 0 'mg'
define D870: ToQuantity('5 \'mg\'').unit
define D871: ToQuantity('5').unit
define D872: ToQuantity(13).unit
define D873: ToQuantity(0.1).unit
define D874: ToQuantity('5 \'days\'').unit
define D875: ToQuantity('1.5e2 \'mg\'').value
define D876: ToQuantity('5.-5 \'mg\'')
define D877: ConvertsToQuantity('5.-5 \'mg\'')
define D878: convert '5.-5 \'mg\'' to Quantity
define D879: ToQuantity('5.+5 \'mg\'')
define D880: ConvertsToQuantity('5.+5 \'mg\'')
define D881: convert '5.+5 \'mg\'' to Quantity
define D882: ToQuantity('5.-5')
define D883: ConvertsToQuantity('5.-5')
define D884: convert '5.-5' to Quantity
define D885: ToQuantity('5.+5')
define D886: ConvertsToQuantity('5.+5')
define D887: convert '5.+5' to Quantity
define D888: ToQuantity('5.e3 \'mg\'')
define D889: ConvertsToQuantity('5.e3 \'mg\'')
define D890: convert '5.e3 \'mg\'' to Quantity
define D891: ToQuantity('5.5e-3 \'mg\'')
define D892: ConvertsToQuantity('5.5e-3 \'mg\'')
define D893: convert '5.5e-3 \'mg\'' to Quantity
