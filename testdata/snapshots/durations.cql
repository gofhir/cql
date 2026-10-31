library Differential version '1.0'
using FHIR version '4.0.1'
include FHIRHelpers version '4.0.1'
// Written by hand, then a grid: the cases of docs/plans/015-years-between-leap.md.
// `years between` and `months between` count whole calendar periods (CQL 1.5.3,
// Appendix B, Duration; Author's Guide, Date and Time Arithmetic), around the
// leap day, the end of a month, the time of day and the edges of the calendar,
// in both directions, with Date and DateTime; the CalculateAgeIn…At operators,
// defined in terms of that duration; and the uncertainty a low precision gives.
// D42 is left out: the reference translator refuses CalculateAgeInYears with two
// arguments. The grid starts at D50.  The gaps in the numbering are definitions
// that were there twice.
define D0: years between @2012-02-29 and @2014-02-28
define D1: years between @2012-02-29 and @2013-02-28
define D2: years between @2012-02-29 and @2013-02-27
define D3: years between @2012-02-29 and @2016-02-28
define D4: years between @2012-02-29 and @2016-02-29
define D5: years between @2012-02-29T12:34:56 and @2014-02-28T12:34:56
define D6: years between @2012-02-29T12:34:56 and @2014-02-28T12:34:55
define D7: years between @2012-02-29T12:34:56 and @2014-02-28T12:34:57
define D8: years between @2014-02-28 and @2012-02-29
define D9: years between @2013-02-28 and @2012-02-29
define D10: years between @2014-02-28T12:34:56 and @2012-02-29T12:34:56
define D11: years between @2011-03-01 and @2012-02-29
define D12: years between @2011-02-28 and @2012-02-29
define D13: years between @2012-03-01 and @2013-02-28
define D14: years between @2012-01-15 and @2014-01-14
define D15: years between @2012-01-15 and @2014-01-15
define D16: years between @2014-01-15 and @2012-01-16
define D17: years between @2012-02-29T23:00:00 and @2014-02-28T22:00:00
define D18: years between @2012-02-29T00:00:00.000Z and @2014-02-28T00:00:00.000+05:00
define D19: months between @2014-01-31 and @2014-02-28
define D20: months between @2014-01-31 and @2014-02-27
define D21: months between @2012-01-31 and @2012-02-29
define D22: months between @2012-01-31 and @2012-02-28
define D23: months between @2014-03-31 and @2014-04-30
define D24: months between @2014-01-30 and @2014-02-28
define D25: months between @2014-02-28 and @2014-01-31
define D26: months between @2014-02-28 and @2014-01-28
define D27: months between @2014-03-31 and @2014-02-28
define D28: months between @2014-01-31T10:00:00 and @2014-02-28T10:00:00
define D29: months between @2014-01-31T10:00:00 and @2014-02-28T09:59:59
define D30: months between @2014-01-15T10:00:00 and @2014-02-15T09:59:59
define D31: years between @2014-01-15T10:00:00 and @2015-01-15T09:59:59
define D32: weeks between @2012-02-29 and @2012-03-07
define D33: days between @2012-02-29 and @2013-02-28
define D34: CalculateAgeInYearsAt(@2012-02-29, @2014-02-28)
define D35: CalculateAgeInYearsAt(@2012-02-29, @2013-02-28)
define D36: CalculateAgeInYearsAt(@2012-02-29, @2013-02-27)
define D37: CalculateAgeInYearsAt(@2000-06-01, @2019-06-01)
define D38: CalculateAgeInMonthsAt(@2014-01-31, @2014-02-28)
define D39: CalculateAgeInMonthsAt(@2014-01-31, @2014-02-27)
define D40: CalculateAgeInYearsAt(@2012-02-29T12:00:00, @2014-02-28T11:00:00)
define D41: CalculateAgeInYearsAt(@2012-02-29T12:00:00, @2014-02-28T12:00:00)
define D43: years between @2012 and @2014
define D44: years between @2012-02 and @2014-02-28
define D45: months between @2014-01 and @2014-02-28
define D46: years between DateTime(2012, 2, 29) and DateTime(2014, 2, 28)
define D47: @2012-02-29 + 2 years
define D48: @2014-01-31 + 1 month
define D49: @2014-02-28 - 1 month
define D50: months between @2013-01-28 and @2013-02-27
define D51: months between @2013-02-27 and @2013-01-28
define D52: months between @2013-01-28 and @2013-02-28
define D53: months between @2013-02-28 and @2013-01-28
define D54: months between @2013-01-29 and @2013-02-27
define D55: months between @2013-02-27 and @2013-01-29
define D56: months between @2013-01-29 and @2013-02-28
define D57: months between @2013-02-28 and @2013-01-29
define D58: months between @2013-01-30 and @2013-02-27
define D59: months between @2013-02-27 and @2013-01-30
define D60: months between @2013-01-30 and @2013-02-28
define D61: months between @2013-02-28 and @2013-01-30
define D62: months between @2013-01-31 and @2013-02-27
define D63: months between @2013-02-27 and @2013-01-31
define D64: months between @2013-01-31 and @2013-02-28
define D65: months between @2013-02-28 and @2013-01-31
define D66: months between @2012-01-29 and @2012-02-27
define D67: months between @2012-02-27 and @2012-01-29
define D68: months between @2012-01-29 and @2012-02-28
define D69: months between @2012-02-28 and @2012-01-29
define D70: months between @2012-01-29 and @2012-02-29
define D71: months between @2012-02-29 and @2012-01-29
define D72: months between @2012-01-30 and @2012-02-27
define D73: months between @2012-02-27 and @2012-01-30
define D74: months between @2012-01-30 and @2012-02-28
define D75: months between @2012-02-28 and @2012-01-30
define D76: months between @2012-01-30 and @2012-02-29
define D77: months between @2012-02-29 and @2012-01-30
define D78: months between @2012-01-31 and @2012-02-27
define D79: months between @2012-02-27 and @2012-01-31
define D80: months between @2012-01-31 and @2012-02-28
define D81: months between @2012-02-28 and @2012-01-31
define D82: months between @2012-01-31 and @2012-02-29
define D83: months between @2012-02-29 and @2012-01-31
define D84: months between @2013-03-30 and @2013-04-27
define D85: months between @2013-04-27 and @2013-03-30
define D86: months between @2013-03-30 and @2013-04-28
define D87: months between @2013-04-28 and @2013-03-30
define D88: months between @2013-03-30 and @2013-04-29
define D89: months between @2013-04-29 and @2013-03-30
define D90: months between @2013-03-30 and @2013-04-30
define D91: months between @2013-04-30 and @2013-03-30
define D92: months between @2013-03-31 and @2013-04-27
define D93: months between @2013-04-27 and @2013-03-31
define D94: months between @2013-03-31 and @2013-04-28
define D95: months between @2013-04-28 and @2013-03-31
define D96: months between @2013-03-31 and @2013-04-29
define D97: months between @2013-04-29 and @2013-03-31
define D98: months between @2013-03-31 and @2013-04-30
define D99: months between @2013-04-30 and @2013-03-31
define D100: months between @2012-03-31 and @2012-04-27
define D101: months between @2012-04-27 and @2012-03-31
define D102: months between @2012-03-31 and @2012-04-28
define D103: months between @2012-04-28 and @2012-03-31
define D104: months between @2012-03-31 and @2012-04-29
define D105: months between @2012-04-29 and @2012-03-31
define D106: months between @2012-03-31 and @2012-04-30
define D107: months between @2012-04-30 and @2012-03-31
define D108: years between @2012-02-29 and @2013-02-27
define D109: years between @2013-02-27 and @2012-02-29
define D110: years between @2012-02-29 and @2013-02-28
define D111: years between @2013-02-28 and @2012-02-29
define D112: years between @2012-02-29 and @2013-03-01
define D113: years between @2013-03-01 and @2012-02-29
define D114: years between @2012-02-29 and @2014-02-27
define D115: years between @2014-02-27 and @2012-02-29
define D116: years between @2012-02-29 and @2014-02-28
define D117: years between @2014-02-28 and @2012-02-29
define D118: years between @2012-02-29 and @2014-03-01
define D119: years between @2014-03-01 and @2012-02-29
define D120: years between @2012-02-29 and @2016-02-27
define D121: years between @2016-02-27 and @2012-02-29
define D122: years between @2012-02-29 and @2016-02-28
define D123: years between @2016-02-28 and @2012-02-29
define D124: years between @2012-02-29 and @2016-02-29
define D125: years between @2016-02-29 and @2012-02-29
define D126: years between @2012-02-29 and @2016-03-01
define D127: years between @2016-03-01 and @2012-02-29
define D128: years between @2011-02-28 and @2013-02-27
define D129: years between @2013-02-27 and @2011-02-28
define D130: years between @2011-02-28 and @2013-02-28
define D131: years between @2013-02-28 and @2011-02-28
define D132: years between @2011-02-28 and @2013-03-01
define D133: years between @2013-03-01 and @2011-02-28
define D134: years between @2011-02-28 and @2014-02-27
define D135: years between @2014-02-27 and @2011-02-28
define D136: years between @2011-02-28 and @2014-02-28
define D137: years between @2014-02-28 and @2011-02-28
define D138: years between @2011-02-28 and @2014-03-01
define D139: years between @2014-03-01 and @2011-02-28
define D140: years between @2011-02-28 and @2016-02-27
define D141: years between @2016-02-27 and @2011-02-28
define D142: years between @2011-02-28 and @2016-02-28
define D143: years between @2016-02-28 and @2011-02-28
define D144: years between @2011-02-28 and @2016-02-29
define D145: years between @2016-02-29 and @2011-02-28
define D146: years between @2011-02-28 and @2016-03-01
define D147: years between @2016-03-01 and @2011-02-28
define D148: years between @2012-02-28 and @2013-02-27
define D149: years between @2013-02-27 and @2012-02-28
define D150: years between @2012-02-28 and @2013-02-28
define D151: years between @2013-02-28 and @2012-02-28
define D152: years between @2012-02-28 and @2013-03-01
define D153: years between @2013-03-01 and @2012-02-28
define D154: years between @2012-02-28 and @2014-02-27
define D155: years between @2014-02-27 and @2012-02-28
define D156: years between @2012-02-28 and @2014-02-28
define D157: years between @2014-02-28 and @2012-02-28
define D158: years between @2012-02-28 and @2014-03-01
define D159: years between @2014-03-01 and @2012-02-28
define D160: years between @2012-02-28 and @2016-02-27
define D161: years between @2016-02-27 and @2012-02-28
define D162: years between @2012-02-28 and @2016-02-28
define D163: years between @2016-02-28 and @2012-02-28
define D164: years between @2012-02-28 and @2016-02-29
define D165: years between @2016-02-29 and @2012-02-28
define D166: years between @2012-02-28 and @2016-03-01
define D167: years between @2016-03-01 and @2012-02-28
define D168: years between @2012-03-01 and @2013-02-27
define D169: years between @2013-02-27 and @2012-03-01
define D170: years between @2012-03-01 and @2013-02-28
define D171: years between @2013-02-28 and @2012-03-01
define D172: years between @2012-03-01 and @2013-03-01
define D173: years between @2013-03-01 and @2012-03-01
define D174: years between @2012-03-01 and @2014-02-27
define D175: years between @2014-02-27 and @2012-03-01
define D176: years between @2012-03-01 and @2014-02-28
define D177: years between @2014-02-28 and @2012-03-01
define D178: years between @2012-03-01 and @2014-03-01
define D179: years between @2014-03-01 and @2012-03-01
define D180: years between @2012-03-01 and @2016-02-27
define D181: years between @2016-02-27 and @2012-03-01
define D182: years between @2012-03-01 and @2016-02-28
define D183: years between @2016-02-28 and @2012-03-01
define D184: years between @2012-03-01 and @2016-02-29
define D185: years between @2016-02-29 and @2012-03-01
define D186: years between @2012-03-01 and @2016-03-01
define D187: years between @2016-03-01 and @2012-03-01
define D188: years between @2011-03-01 and @2013-02-27
define D189: years between @2013-02-27 and @2011-03-01
define D190: years between @2011-03-01 and @2013-02-28
define D191: years between @2013-02-28 and @2011-03-01
define D192: years between @2011-03-01 and @2013-03-01
define D193: years between @2013-03-01 and @2011-03-01
define D194: years between @2011-03-01 and @2014-02-27
define D195: years between @2014-02-27 and @2011-03-01
define D196: years between @2011-03-01 and @2014-02-28
define D197: years between @2014-02-28 and @2011-03-01
define D198: years between @2011-03-01 and @2014-03-01
define D199: years between @2014-03-01 and @2011-03-01
define D200: years between @2011-03-01 and @2016-02-27
define D201: years between @2016-02-27 and @2011-03-01
define D202: years between @2011-03-01 and @2016-02-28
define D203: years between @2016-02-28 and @2011-03-01
define D204: years between @2011-03-01 and @2016-02-29
define D205: years between @2016-02-29 and @2011-03-01
define D206: years between @2011-03-01 and @2016-03-01
define D207: years between @2016-03-01 and @2011-03-01
define D208: years between @2012-02-29T12:34:56 and @2013-02-28T12:34:56
define D209: years between @2013-02-28T12:34:56 and @2012-02-29T12:34:56
define D210: years between @2012-02-29T12:34:56 and @2013-02-28T12:34:55
define D211: years between @2013-02-28T12:34:55 and @2012-02-29T12:34:56
define D212: years between @2012-02-29T12:34:56 and @2013-02-28T12:34:57
define D213: years between @2013-02-28T12:34:57 and @2012-02-29T12:34:56
define D214: years between @2012-02-29T12:34:56 and @2014-02-28T12:34:56
define D215: years between @2014-02-28T12:34:56 and @2012-02-29T12:34:56
define D216: years between @2012-02-29T12:34:56 and @2014-02-28T12:34:55
define D217: years between @2014-02-28T12:34:55 and @2012-02-29T12:34:56
define D218: years between @2012-02-29T12:34:56 and @2014-02-28T12:34:57
define D219: years between @2014-02-28T12:34:57 and @2012-02-29T12:34:56
define D220: years between @2012-02-29T12:34:56 and @2016-02-29T12:34:56
define D221: years between @2016-02-29T12:34:56 and @2012-02-29T12:34:56
define D222: years between @2012-02-29T12:34:56 and @2016-02-29T12:34:55
define D223: years between @2016-02-29T12:34:55 and @2012-02-29T12:34:56
define D224: years between @2012-02-29T12:34:56 and @2016-02-29T12:34:57
define D225: years between @2016-02-29T12:34:57 and @2012-02-29T12:34:56
define D226: years between @2011-02-28T12:34:56 and @2012-02-29T12:34:56
define D227: years between @2012-02-29T12:34:56 and @2011-02-28T12:34:56
define D228: years between @2011-02-28T12:34:56 and @2012-02-29T12:34:55
define D229: years between @2012-02-29T12:34:55 and @2011-02-28T12:34:56
define D230: years between @2011-02-28T12:34:56 and @2012-02-29T12:34:57
define D231: years between @2012-02-29T12:34:57 and @2011-02-28T12:34:56
define D232: years between @2011-02-28T12:34:56 and @2013-02-28T12:34:56
define D233: years between @2013-02-28T12:34:56 and @2011-02-28T12:34:56
define D234: years between @2011-02-28T12:34:56 and @2013-02-28T12:34:55
define D235: years between @2013-02-28T12:34:55 and @2011-02-28T12:34:56
define D236: years between @2011-02-28T12:34:56 and @2013-02-28T12:34:57
define D237: years between @2013-02-28T12:34:57 and @2011-02-28T12:34:56
define D238: months between @2013-01-31T10:00:00 and @2013-02-28T10:00:00
define D239: months between @2013-02-28T10:00:00 and @2013-01-31T10:00:00
define D240: months between @2013-01-31T10:00:00 and @2013-02-28T09:59:59
define D241: months between @2013-02-28T09:59:59 and @2013-01-31T10:00:00
define D242: months between @2013-01-31T10:00:00 and @2013-02-28T10:00:01
define D243: months between @2013-02-28T10:00:01 and @2013-01-31T10:00:00
define D244: months between @2013-01-31T10:00:00 and @2013-03-31T10:00:00
define D245: months between @2013-03-31T10:00:00 and @2013-01-31T10:00:00
define D246: months between @2013-01-31T10:00:00 and @2013-03-31T09:59:59
define D247: months between @2013-03-31T09:59:59 and @2013-01-31T10:00:00
define D248: months between @2013-01-31T10:00:00 and @2013-03-31T10:00:01
define D249: months between @2013-03-31T10:00:01 and @2013-01-31T10:00:00
define D250: months between @2012-01-31T10:00:00 and @2012-02-29T10:00:00
define D251: months between @2012-02-29T10:00:00 and @2012-01-31T10:00:00
define D252: months between @2012-01-31T10:00:00 and @2012-02-29T09:59:59
define D253: months between @2012-02-29T09:59:59 and @2012-01-31T10:00:00
define D254: months between @2012-01-31T10:00:00 and @2012-02-29T10:00:01
define D255: months between @2012-02-29T10:00:01 and @2012-01-31T10:00:00
define D256: months between @2012-01-31T10:00:00 and @2012-03-31T10:00:00
define D257: months between @2012-03-31T10:00:00 and @2012-01-31T10:00:00
define D258: months between @2012-01-31T10:00:00 and @2012-03-31T09:59:59
define D259: months between @2012-03-31T09:59:59 and @2012-01-31T10:00:00
define D260: months between @2012-01-31T10:00:00 and @2012-03-31T10:00:01
define D261: months between @2012-03-31T10:00:01 and @2012-01-31T10:00:00
define D262: years between @2012-02-29T12:34:56.000 and @2014-02-28T12:34:56.000
define D263: years between @2012-02-29T12:34:56.001 and @2014-02-28T12:34:56.000
define D264: years between @2012-02-29T12:34:56.000 and @2014-02-28T12:34:56.001
define D265: months between @2014-01-31T10:00:00.000 and @2014-02-28T09:59:59.999
define D266: years between @2012-02-29 and @2014-02-28T00:00:00.000
define D267: years between @2012-02-29T12:00 and @2014-02-28T11:59:59.999
define D268: years between @2012-02-29T12:00:00 and @2014-02-28T12:00
define D269: months between @2014-01-31 and @2014-02
define D270: years between @0001-01-01 and @0002-01-01
define D271: years between @0002-01-01 and @0001-01-01
define D272: months between @0001-01-01 and @0001-02-01
define D273: years between @0001-03-01 and @9999-02-28
define D274: years between @9999-12-31 and @0001-02-28
define D275: months between @9999-01-31 and @9999-02-28
define D276: years between @2014 and @2012
define D277: years between @2012 and @2014
define D278: months between @2014-03 and @2014-02
define D279: months between @2014-02 and @2014-03
define D280: years between DateTime(2014) and DateTime(2012)
define D281: months between DateTime(2005) and DateTime(2006, 7)
define D282: CalculateAgeInYearsAt(@2012-02-29, @2014-02-28)
define D283: CalculateAgeInYearsAt(@2012-02-29, @2013-02-28)
define D284: CalculateAgeInYearsAt(@2012-02-29, @2016-02-28)
define D285: CalculateAgeInYearsAt(@2012-02-29, @2016-02-29)
define D286: CalculateAgeInYearsAt(@2014-02-28, @2012-02-29)
define D287: CalculateAgeInYearsAt(@2011-02-28, @2012-02-29)
define D288: CalculateAgeInMonthsAt(@2013-01-31, @2013-02-28)
define D289: CalculateAgeInMonthsAt(@2012-01-31, @2012-02-29)
define D290: CalculateAgeInMonthsAt(@2012-01-31, @2012-02-28)
define D291: CalculateAgeInMonthsAt(@2013-03-31, @2013-04-30)
define D292: CalculateAgeInMonthsAt(@2013-02-28, @2013-01-31)
define D293: CalculateAgeInYearsAt(@2012-02-29T12:00:00, @2014-02-28T11:00:00)
define D294: CalculateAgeInMonthsAt(@2014-01-31T10:00:00, @2014-02-28T09:00:00)
define D295: CalculateAgeInWeeksAt(@2012-02-29, @2012-03-07)
define D296: CalculateAgeInDaysAt(@2012-02-29, @2013-02-28)
define D297: CalculateAgeInYearsAt(@2012-02-29, @2014-02-28T00:00:00.000)

// A tuple whose fields are those of an uncertainty is still a tuple (review 015,
// round 1): the JS encoder read the field rather than the class.
define D298: Tuple { isUncertainty: true, low: 1, high: 2 }
