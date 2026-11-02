library Differential version '1.0'
using FHIR version '4.0.1'
include FHIRHelpers version '4.0.1'
// Written by hand, then a grid: the cases of docs/plans/016-duration-uncertainty.md.
// An uncertainty is a point of unknown value: a comparison with it is true if it
// holds for every value it could be, false if it holds for none, and null
// otherwise (CQL 1.5.3, Language Semantics, Uncertainty). The normative examples of
// Determining Difference and Duration and the rule that seconds and milliseconds
// are one precision come first; then each base uncertainty against points, closed
// and open intervals (in, during, included in, includes, contains, between), lists
// and other uncertainties.
define D0: days between @2014-01-15 and @2014-02
define D1: days between @2017-08-07T17:00 and @2017-08-14T
define D2: days between @2012-01 and @2012-02
define D3: years between DateTime(2005) and DateTime(2010)
define D4: days between DateTime(2012, 1, 1) and DateTime(2012, 1, 3)
define D5: days between @2017-08-07T and @2017-08-14T17:00
define D6: days between @2017-08-14T and @2017-08-07T17:00
define D7: days between @2017-08-07T17:00:00 and @2017-08-14T
define D8: hours between @2012-01-01T01:00:00 and @2012-01-01T02:00:00.0
define D9: seconds between @2012-01-01T01:00:00 and @2012-01-01T01:00:01.0
define D10: seconds between @2012-01-01T01:00:00.0 and @2012-01-01T01:00:01
define D11: seconds between @2012-01-01T01:00:00 and @2012-01-01T01:00:01
define D12: milliseconds between @2012-01-01T01:00:00 and @2012-01-01T01:00:01
define D13: seconds between @T01:00:00 and @T01:00:01.0
define D14: seconds between @2012-01-01T01:00 and @2012-01-01T01:01
define D15: minutes between @2012-01-01T01 and @2012-01-01T03
define D16: (days between @2014-01-15 and @2014-02) > 2
define D17: (days between @2014-01-15 and @2014-02) > 50
define D18: (days between @2014-01-15 and @2014-02) > 20
define D19: (days between @2014-01-15 and @2014-02) < 50
define D20: (days between @2014-01-15 and @2014-02) < 2
define D21: (days between @2014-01-15 and @2014-02) < 20
define D22: (days between @2014-01-15 and @2014-02) >= 17
define D23: (days between @2014-01-15 and @2014-02) <= 44
define D24: (days between @2014-01-15 and @2014-02) >= 18
define D25: (days between @2014-01-15 and @2014-02) <= 43
define D26: (years between @2012 and @2014) < 0
define D27: (years between @2012 and @2014) <= 0
define D28: (years between @2012 and @2014) > 0
define D29: (years between @2012 and @2014) >= 0
define D30: (years between @2012 and @2014) = 0
define D31: (years between @2012 and @2014) before 0
define D32: (years between @2012 and @2014) after 0
define D33: (years between @2012 and @2014) same or before 0
define D34: (years between @2012 and @2014) same or after 0
define D35: (years between @2012 and @2014) same as 0
define D36: (years between @2012 and @2014) < 1
define D37: (years between @2012 and @2014) <= 1
define D38: (years between @2012 and @2014) > 1
define D39: (years between @2012 and @2014) >= 1
define D40: (years between @2012 and @2014) = 1
define D41: (years between @2012 and @2014) before 1
define D42: (years between @2012 and @2014) after 1
define D43: (years between @2012 and @2014) same or before 1
define D44: (years between @2012 and @2014) same or after 1
define D45: (years between @2012 and @2014) same as 1
define D46: (years between @2012 and @2014) < 2
define D47: (years between @2012 and @2014) <= 2
define D48: (years between @2012 and @2014) > 2
define D49: (years between @2012 and @2014) >= 2
define D50: (years between @2012 and @2014) = 2
define D51: (years between @2012 and @2014) before 2
define D52: (years between @2012 and @2014) after 2
define D53: (years between @2012 and @2014) same or before 2
define D54: (years between @2012 and @2014) same or after 2
define D55: (years between @2012 and @2014) same as 2
define D56: (years between @2012 and @2014) < 3
define D57: (years between @2012 and @2014) <= 3
define D58: (years between @2012 and @2014) > 3
define D59: (years between @2012 and @2014) >= 3
define D60: (years between @2012 and @2014) = 3
define D61: (years between @2012 and @2014) before 3
define D62: (years between @2012 and @2014) after 3
define D63: (years between @2012 and @2014) same or before 3
define D64: (years between @2012 and @2014) same or after 3
define D65: (years between @2012 and @2014) same as 3
define D66: (years between @2012 and @2014) in Interval[0, 3]
define D67: (years between @2012 and @2014) during Interval[0, 3]
define D68: (years between @2012 and @2014) included in Interval[0, 3]
define D69: Interval[0, 3] includes (years between @2012 and @2014)
define D70: Interval[0, 3] contains (years between @2012 and @2014)
define D71: (years between @2012 and @2014) in Interval(0, 3)
define D72: (years between @2012 and @2014) during Interval(0, 3)
define D73: (years between @2012 and @2014) included in Interval(0, 3)
define D74: Interval(0, 3) includes (years between @2012 and @2014)
define D75: Interval(0, 3) contains (years between @2012 and @2014)
define D76: (years between @2012 and @2014) in Interval[0, 3)
define D77: (years between @2012 and @2014) during Interval[0, 3)
define D78: (years between @2012 and @2014) included in Interval[0, 3)
define D79: Interval[0, 3) includes (years between @2012 and @2014)
define D80: Interval[0, 3) contains (years between @2012 and @2014)
define D81: (years between @2012 and @2014) in Interval(0, 3]
define D82: (years between @2012 and @2014) during Interval(0, 3]
define D83: (years between @2012 and @2014) included in Interval(0, 3]
define D84: Interval(0, 3] includes (years between @2012 and @2014)
define D85: Interval(0, 3] contains (years between @2012 and @2014)
define D86: (years between @2012 and @2014) between 0 and 3
define D87: (years between @2012 and @2014) properly between 0 and 3
define D88: (years between @2012 and @2014) in Interval[1, 3]
define D89: (years between @2012 and @2014) during Interval[1, 3]
define D90: (years between @2012 and @2014) included in Interval[1, 3]
define D91: Interval[1, 3] includes (years between @2012 and @2014)
define D92: Interval[1, 3] contains (years between @2012 and @2014)
define D93: (years between @2012 and @2014) in Interval(1, 3)
define D94: (years between @2012 and @2014) during Interval(1, 3)
define D95: (years between @2012 and @2014) included in Interval(1, 3)
define D96: Interval(1, 3) includes (years between @2012 and @2014)
define D97: Interval(1, 3) contains (years between @2012 and @2014)
define D98: (years between @2012 and @2014) in Interval[1, 3)
define D99: (years between @2012 and @2014) during Interval[1, 3)
define D100: (years between @2012 and @2014) included in Interval[1, 3)
define D101: Interval[1, 3) includes (years between @2012 and @2014)
define D102: Interval[1, 3) contains (years between @2012 and @2014)
define D103: (years between @2012 and @2014) in Interval(1, 3]
define D104: (years between @2012 and @2014) during Interval(1, 3]
define D105: (years between @2012 and @2014) included in Interval(1, 3]
define D106: Interval(1, 3] includes (years between @2012 and @2014)
define D107: Interval(1, 3] contains (years between @2012 and @2014)
define D108: (years between @2012 and @2014) between 1 and 3
define D109: (years between @2012 and @2014) properly between 1 and 3
define D110: (years between @2012 and @2014) in Interval[0, 2]
define D111: (years between @2012 and @2014) during Interval[0, 2]
define D112: (years between @2012 and @2014) included in Interval[0, 2]
define D113: Interval[0, 2] includes (years between @2012 and @2014)
define D114: Interval[0, 2] contains (years between @2012 and @2014)
define D115: (years between @2012 and @2014) in Interval(0, 2)
define D116: (years between @2012 and @2014) during Interval(0, 2)
define D117: (years between @2012 and @2014) included in Interval(0, 2)
define D118: Interval(0, 2) includes (years between @2012 and @2014)
define D119: Interval(0, 2) contains (years between @2012 and @2014)
define D120: (years between @2012 and @2014) in Interval[0, 2)
define D121: (years between @2012 and @2014) during Interval[0, 2)
define D122: (years between @2012 and @2014) included in Interval[0, 2)
define D123: Interval[0, 2) includes (years between @2012 and @2014)
define D124: Interval[0, 2) contains (years between @2012 and @2014)
define D125: (years between @2012 and @2014) in Interval(0, 2]
define D126: (years between @2012 and @2014) during Interval(0, 2]
define D127: (years between @2012 and @2014) included in Interval(0, 2]
define D128: Interval(0, 2] includes (years between @2012 and @2014)
define D129: Interval(0, 2] contains (years between @2012 and @2014)
define D130: (years between @2012 and @2014) between 0 and 2
define D131: (years between @2012 and @2014) properly between 0 and 2
define D132: (years between @2012 and @2014) in Interval[1, 3]
define D133: (years between @2012 and @2014) during Interval[1, 3]
define D134: (years between @2012 and @2014) included in Interval[1, 3]
define D135: Interval[1, 3] includes (years between @2012 and @2014)
define D136: Interval[1, 3] contains (years between @2012 and @2014)
define D137: (years between @2012 and @2014) in Interval(1, 3)
define D138: (years between @2012 and @2014) during Interval(1, 3)
define D139: (years between @2012 and @2014) included in Interval(1, 3)
define D140: Interval(1, 3) includes (years between @2012 and @2014)
define D141: Interval(1, 3) contains (years between @2012 and @2014)
define D142: (years between @2012 and @2014) in Interval[1, 3)
define D143: (years between @2012 and @2014) during Interval[1, 3)
define D144: (years between @2012 and @2014) included in Interval[1, 3)
define D145: Interval[1, 3) includes (years between @2012 and @2014)
define D146: Interval[1, 3) contains (years between @2012 and @2014)
define D147: (years between @2012 and @2014) in Interval(1, 3]
define D148: (years between @2012 and @2014) during Interval(1, 3]
define D149: (years between @2012 and @2014) included in Interval(1, 3]
define D150: Interval(1, 3] includes (years between @2012 and @2014)
define D151: Interval(1, 3] contains (years between @2012 and @2014)
define D152: (years between @2012 and @2014) between 1 and 3
define D153: (years between @2012 and @2014) properly between 1 and 3
define D154: (years between @2012 and @2014) in Interval[2, 4]
define D155: (years between @2012 and @2014) during Interval[2, 4]
define D156: (years between @2012 and @2014) included in Interval[2, 4]
define D157: Interval[2, 4] includes (years between @2012 and @2014)
define D158: Interval[2, 4] contains (years between @2012 and @2014)
define D159: (years between @2012 and @2014) in Interval(2, 4)
define D160: (years between @2012 and @2014) during Interval(2, 4)
define D161: (years between @2012 and @2014) included in Interval(2, 4)
define D162: Interval(2, 4) includes (years between @2012 and @2014)
define D163: Interval(2, 4) contains (years between @2012 and @2014)
define D164: (years between @2012 and @2014) in Interval[2, 4)
define D165: (years between @2012 and @2014) during Interval[2, 4)
define D166: (years between @2012 and @2014) included in Interval[2, 4)
define D167: Interval[2, 4) includes (years between @2012 and @2014)
define D168: Interval[2, 4) contains (years between @2012 and @2014)
define D169: (years between @2012 and @2014) in Interval(2, 4]
define D170: (years between @2012 and @2014) during Interval(2, 4]
define D171: (years between @2012 and @2014) included in Interval(2, 4]
define D172: Interval(2, 4] includes (years between @2012 and @2014)
define D173: Interval(2, 4] contains (years between @2012 and @2014)
define D174: (years between @2012 and @2014) between 2 and 4
define D175: (years between @2012 and @2014) properly between 2 and 4
define D176: (years between @2012 and @2014) in { 1, 2 }
define D177: (years between @2012 and @2014) in { 5, 6 }
define D178: { 1, 5 } contains (years between @2012 and @2014)
define D179: (years between @2014 and @2012) < (-3)
define D180: (years between @2014 and @2012) <= (-3)
define D181: (years between @2014 and @2012) > (-3)
define D182: (years between @2014 and @2012) >= (-3)
define D183: (years between @2014 and @2012) = (-3)
define D184: (years between @2014 and @2012) before (-3)
define D185: (years between @2014 and @2012) after (-3)
define D186: (years between @2014 and @2012) same or before (-3)
define D187: (years between @2014 and @2012) same or after (-3)
define D188: (years between @2014 and @2012) same as (-3)
define D189: (years between @2014 and @2012) < (-2)
define D190: (years between @2014 and @2012) <= (-2)
define D191: (years between @2014 and @2012) > (-2)
define D192: (years between @2014 and @2012) >= (-2)
define D193: (years between @2014 and @2012) = (-2)
define D194: (years between @2014 and @2012) before (-2)
define D195: (years between @2014 and @2012) after (-2)
define D196: (years between @2014 and @2012) same or before (-2)
define D197: (years between @2014 and @2012) same or after (-2)
define D198: (years between @2014 and @2012) same as (-2)
define D199: (years between @2014 and @2012) < (-1)
define D200: (years between @2014 and @2012) <= (-1)
define D201: (years between @2014 and @2012) > (-1)
define D202: (years between @2014 and @2012) >= (-1)
define D203: (years between @2014 and @2012) = (-1)
define D204: (years between @2014 and @2012) before (-1)
define D205: (years between @2014 and @2012) after (-1)
define D206: (years between @2014 and @2012) same or before (-1)
define D207: (years between @2014 and @2012) same or after (-1)
define D208: (years between @2014 and @2012) same as (-1)
define D209: (years between @2014 and @2012) < 0
define D210: (years between @2014 and @2012) <= 0
define D211: (years between @2014 and @2012) > 0
define D212: (years between @2014 and @2012) >= 0
define D213: (years between @2014 and @2012) = 0
define D214: (years between @2014 and @2012) before 0
define D215: (years between @2014 and @2012) after 0
define D216: (years between @2014 and @2012) same or before 0
define D217: (years between @2014 and @2012) same or after 0
define D218: (years between @2014 and @2012) same as 0
define D219: (years between @2014 and @2012) in Interval[(-3), 0]
define D220: (years between @2014 and @2012) during Interval[(-3), 0]
define D221: (years between @2014 and @2012) included in Interval[(-3), 0]
define D222: Interval[(-3), 0] includes (years between @2014 and @2012)
define D223: Interval[(-3), 0] contains (years between @2014 and @2012)
define D224: (years between @2014 and @2012) in Interval((-3), 0)
define D225: (years between @2014 and @2012) during Interval((-3), 0)
define D226: (years between @2014 and @2012) included in Interval((-3), 0)
define D227: Interval((-3), 0) includes (years between @2014 and @2012)
define D228: Interval((-3), 0) contains (years between @2014 and @2012)
define D229: (years between @2014 and @2012) in Interval[(-3), 0)
define D230: (years between @2014 and @2012) during Interval[(-3), 0)
define D231: (years between @2014 and @2012) included in Interval[(-3), 0)
define D232: Interval[(-3), 0) includes (years between @2014 and @2012)
define D233: Interval[(-3), 0) contains (years between @2014 and @2012)
define D234: (years between @2014 and @2012) in Interval((-3), 0]
define D235: (years between @2014 and @2012) during Interval((-3), 0]
define D236: (years between @2014 and @2012) included in Interval((-3), 0]
define D237: Interval((-3), 0] includes (years between @2014 and @2012)
define D238: Interval((-3), 0] contains (years between @2014 and @2012)
define D239: (years between @2014 and @2012) between (-3) and 0
define D240: (years between @2014 and @2012) properly between (-3) and 0
define D241: (years between @2014 and @2012) in Interval[(-2), 0]
define D242: (years between @2014 and @2012) during Interval[(-2), 0]
define D243: (years between @2014 and @2012) included in Interval[(-2), 0]
define D244: Interval[(-2), 0] includes (years between @2014 and @2012)
define D245: Interval[(-2), 0] contains (years between @2014 and @2012)
define D246: (years between @2014 and @2012) in Interval((-2), 0)
define D247: (years between @2014 and @2012) during Interval((-2), 0)
define D248: (years between @2014 and @2012) included in Interval((-2), 0)
define D249: Interval((-2), 0) includes (years between @2014 and @2012)
define D250: Interval((-2), 0) contains (years between @2014 and @2012)
define D251: (years between @2014 and @2012) in Interval[(-2), 0)
define D252: (years between @2014 and @2012) during Interval[(-2), 0)
define D253: (years between @2014 and @2012) included in Interval[(-2), 0)
define D254: Interval[(-2), 0) includes (years between @2014 and @2012)
define D255: Interval[(-2), 0) contains (years between @2014 and @2012)
define D256: (years between @2014 and @2012) in Interval((-2), 0]
define D257: (years between @2014 and @2012) during Interval((-2), 0]
define D258: (years between @2014 and @2012) included in Interval((-2), 0]
define D259: Interval((-2), 0] includes (years between @2014 and @2012)
define D260: Interval((-2), 0] contains (years between @2014 and @2012)
define D261: (years between @2014 and @2012) between (-2) and 0
define D262: (years between @2014 and @2012) properly between (-2) and 0
define D263: (years between @2014 and @2012) in Interval[(-3), (-1)]
define D264: (years between @2014 and @2012) during Interval[(-3), (-1)]
define D265: (years between @2014 and @2012) included in Interval[(-3), (-1)]
define D266: Interval[(-3), (-1)] includes (years between @2014 and @2012)
define D267: Interval[(-3), (-1)] contains (years between @2014 and @2012)
define D268: (years between @2014 and @2012) in Interval((-3), (-1))
define D269: (years between @2014 and @2012) during Interval((-3), (-1))
define D270: (years between @2014 and @2012) included in Interval((-3), (-1))
define D271: Interval((-3), (-1)) includes (years between @2014 and @2012)
define D272: Interval((-3), (-1)) contains (years between @2014 and @2012)
define D273: (years between @2014 and @2012) in Interval[(-3), (-1))
define D274: (years between @2014 and @2012) during Interval[(-3), (-1))
define D275: (years between @2014 and @2012) included in Interval[(-3), (-1))
define D276: Interval[(-3), (-1)) includes (years between @2014 and @2012)
define D277: Interval[(-3), (-1)) contains (years between @2014 and @2012)
define D278: (years between @2014 and @2012) in Interval((-3), (-1)]
define D279: (years between @2014 and @2012) during Interval((-3), (-1)]
define D280: (years between @2014 and @2012) included in Interval((-3), (-1)]
define D281: Interval((-3), (-1)] includes (years between @2014 and @2012)
define D282: Interval((-3), (-1)] contains (years between @2014 and @2012)
define D283: (years between @2014 and @2012) between (-3) and (-1)
define D284: (years between @2014 and @2012) properly between (-3) and (-1)
define D285: (years between @2014 and @2012) in Interval[(-2), 0]
define D286: (years between @2014 and @2012) during Interval[(-2), 0]
define D287: (years between @2014 and @2012) included in Interval[(-2), 0]
define D288: Interval[(-2), 0] includes (years between @2014 and @2012)
define D289: Interval[(-2), 0] contains (years between @2014 and @2012)
define D290: (years between @2014 and @2012) in Interval((-2), 0)
define D291: (years between @2014 and @2012) during Interval((-2), 0)
define D292: (years between @2014 and @2012) included in Interval((-2), 0)
define D293: Interval((-2), 0) includes (years between @2014 and @2012)
define D294: Interval((-2), 0) contains (years between @2014 and @2012)
define D295: (years between @2014 and @2012) in Interval[(-2), 0)
define D296: (years between @2014 and @2012) during Interval[(-2), 0)
define D297: (years between @2014 and @2012) included in Interval[(-2), 0)
define D298: Interval[(-2), 0) includes (years between @2014 and @2012)
define D299: Interval[(-2), 0) contains (years between @2014 and @2012)
define D300: (years between @2014 and @2012) in Interval((-2), 0]
define D301: (years between @2014 and @2012) during Interval((-2), 0]
define D302: (years between @2014 and @2012) included in Interval((-2), 0]
define D303: Interval((-2), 0] includes (years between @2014 and @2012)
define D304: Interval((-2), 0] contains (years between @2014 and @2012)
define D305: (years between @2014 and @2012) between (-2) and 0
define D306: (years between @2014 and @2012) properly between (-2) and 0
define D307: (years between @2014 and @2012) in Interval[(-1), 1]
define D308: (years between @2014 and @2012) during Interval[(-1), 1]
define D309: (years between @2014 and @2012) included in Interval[(-1), 1]
define D310: Interval[(-1), 1] includes (years between @2014 and @2012)
define D311: Interval[(-1), 1] contains (years between @2014 and @2012)
define D312: (years between @2014 and @2012) in Interval((-1), 1)
define D313: (years between @2014 and @2012) during Interval((-1), 1)
define D314: (years between @2014 and @2012) included in Interval((-1), 1)
define D315: Interval((-1), 1) includes (years between @2014 and @2012)
define D316: Interval((-1), 1) contains (years between @2014 and @2012)
define D317: (years between @2014 and @2012) in Interval[(-1), 1)
define D318: (years between @2014 and @2012) during Interval[(-1), 1)
define D319: (years between @2014 and @2012) included in Interval[(-1), 1)
define D320: Interval[(-1), 1) includes (years between @2014 and @2012)
define D321: Interval[(-1), 1) contains (years between @2014 and @2012)
define D322: (years between @2014 and @2012) in Interval((-1), 1]
define D323: (years between @2014 and @2012) during Interval((-1), 1]
define D324: (years between @2014 and @2012) included in Interval((-1), 1]
define D325: Interval((-1), 1] includes (years between @2014 and @2012)
define D326: Interval((-1), 1] contains (years between @2014 and @2012)
define D327: (years between @2014 and @2012) between (-1) and 1
define D328: (years between @2014 and @2012) properly between (-1) and 1
define D329: (years between @2014 and @2012) in { (-2), (-1) }
define D330: (years between @2014 and @2012) in { 2, 3 }
define D331: { (-2), 2 } contains (years between @2014 and @2012)
define D332: (years between DateTime(2005) and DateTime(2010)) < 3
define D333: (years between DateTime(2005) and DateTime(2010)) <= 3
define D334: (years between DateTime(2005) and DateTime(2010)) > 3
define D335: (years between DateTime(2005) and DateTime(2010)) >= 3
define D336: (years between DateTime(2005) and DateTime(2010)) = 3
define D337: (years between DateTime(2005) and DateTime(2010)) before 3
define D338: (years between DateTime(2005) and DateTime(2010)) after 3
define D339: (years between DateTime(2005) and DateTime(2010)) same or before 3
define D340: (years between DateTime(2005) and DateTime(2010)) same or after 3
define D341: (years between DateTime(2005) and DateTime(2010)) same as 3
define D342: (years between DateTime(2005) and DateTime(2010)) < 4
define D343: (years between DateTime(2005) and DateTime(2010)) <= 4
define D344: (years between DateTime(2005) and DateTime(2010)) > 4
define D345: (years between DateTime(2005) and DateTime(2010)) >= 4
define D346: (years between DateTime(2005) and DateTime(2010)) = 4
define D347: (years between DateTime(2005) and DateTime(2010)) before 4
define D348: (years between DateTime(2005) and DateTime(2010)) after 4
define D349: (years between DateTime(2005) and DateTime(2010)) same or before 4
define D350: (years between DateTime(2005) and DateTime(2010)) same or after 4
define D351: (years between DateTime(2005) and DateTime(2010)) same as 4
define D352: (years between DateTime(2005) and DateTime(2010)) < 5
define D353: (years between DateTime(2005) and DateTime(2010)) <= 5
define D354: (years between DateTime(2005) and DateTime(2010)) > 5
define D355: (years between DateTime(2005) and DateTime(2010)) >= 5
define D356: (years between DateTime(2005) and DateTime(2010)) = 5
define D357: (years between DateTime(2005) and DateTime(2010)) before 5
define D358: (years between DateTime(2005) and DateTime(2010)) after 5
define D359: (years between DateTime(2005) and DateTime(2010)) same or before 5
define D360: (years between DateTime(2005) and DateTime(2010)) same or after 5
define D361: (years between DateTime(2005) and DateTime(2010)) same as 5
define D362: (years between DateTime(2005) and DateTime(2010)) < 6
define D363: (years between DateTime(2005) and DateTime(2010)) <= 6
define D364: (years between DateTime(2005) and DateTime(2010)) > 6
define D365: (years between DateTime(2005) and DateTime(2010)) >= 6
define D366: (years between DateTime(2005) and DateTime(2010)) = 6
define D367: (years between DateTime(2005) and DateTime(2010)) before 6
define D368: (years between DateTime(2005) and DateTime(2010)) after 6
define D369: (years between DateTime(2005) and DateTime(2010)) same or before 6
define D370: (years between DateTime(2005) and DateTime(2010)) same or after 6
define D371: (years between DateTime(2005) and DateTime(2010)) same as 6
define D372: (years between DateTime(2005) and DateTime(2010)) in Interval[3, 6]
define D373: (years between DateTime(2005) and DateTime(2010)) during Interval[3, 6]
define D374: (years between DateTime(2005) and DateTime(2010)) included in Interval[3, 6]
define D375: Interval[3, 6] includes (years between DateTime(2005) and DateTime(2010))
define D376: Interval[3, 6] contains (years between DateTime(2005) and DateTime(2010))
define D377: (years between DateTime(2005) and DateTime(2010)) in Interval(3, 6)
define D378: (years between DateTime(2005) and DateTime(2010)) during Interval(3, 6)
define D379: (years between DateTime(2005) and DateTime(2010)) included in Interval(3, 6)
define D380: Interval(3, 6) includes (years between DateTime(2005) and DateTime(2010))
define D381: Interval(3, 6) contains (years between DateTime(2005) and DateTime(2010))
define D382: (years between DateTime(2005) and DateTime(2010)) in Interval[3, 6)
define D383: (years between DateTime(2005) and DateTime(2010)) during Interval[3, 6)
define D384: (years between DateTime(2005) and DateTime(2010)) included in Interval[3, 6)
define D385: Interval[3, 6) includes (years between DateTime(2005) and DateTime(2010))
define D386: Interval[3, 6) contains (years between DateTime(2005) and DateTime(2010))
define D387: (years between DateTime(2005) and DateTime(2010)) in Interval(3, 6]
define D388: (years between DateTime(2005) and DateTime(2010)) during Interval(3, 6]
define D389: (years between DateTime(2005) and DateTime(2010)) included in Interval(3, 6]
define D390: Interval(3, 6] includes (years between DateTime(2005) and DateTime(2010))
define D391: Interval(3, 6] contains (years between DateTime(2005) and DateTime(2010))
define D392: (years between DateTime(2005) and DateTime(2010)) between 3 and 6
define D393: (years between DateTime(2005) and DateTime(2010)) properly between 3 and 6
define D394: (years between DateTime(2005) and DateTime(2010)) in Interval[4, 6]
define D395: (years between DateTime(2005) and DateTime(2010)) during Interval[4, 6]
define D396: (years between DateTime(2005) and DateTime(2010)) included in Interval[4, 6]
define D397: Interval[4, 6] includes (years between DateTime(2005) and DateTime(2010))
define D398: Interval[4, 6] contains (years between DateTime(2005) and DateTime(2010))
define D399: (years between DateTime(2005) and DateTime(2010)) in Interval(4, 6)
define D400: (years between DateTime(2005) and DateTime(2010)) during Interval(4, 6)
define D401: (years between DateTime(2005) and DateTime(2010)) included in Interval(4, 6)
define D402: Interval(4, 6) includes (years between DateTime(2005) and DateTime(2010))
define D403: Interval(4, 6) contains (years between DateTime(2005) and DateTime(2010))
define D404: (years between DateTime(2005) and DateTime(2010)) in Interval[4, 6)
define D405: (years between DateTime(2005) and DateTime(2010)) during Interval[4, 6)
define D406: (years between DateTime(2005) and DateTime(2010)) included in Interval[4, 6)
define D407: Interval[4, 6) includes (years between DateTime(2005) and DateTime(2010))
define D408: Interval[4, 6) contains (years between DateTime(2005) and DateTime(2010))
define D409: (years between DateTime(2005) and DateTime(2010)) in Interval(4, 6]
define D410: (years between DateTime(2005) and DateTime(2010)) during Interval(4, 6]
define D411: (years between DateTime(2005) and DateTime(2010)) included in Interval(4, 6]
define D412: Interval(4, 6] includes (years between DateTime(2005) and DateTime(2010))
define D413: Interval(4, 6] contains (years between DateTime(2005) and DateTime(2010))
define D414: (years between DateTime(2005) and DateTime(2010)) between 4 and 6
define D415: (years between DateTime(2005) and DateTime(2010)) properly between 4 and 6
define D416: (years between DateTime(2005) and DateTime(2010)) in Interval[3, 5]
define D417: (years between DateTime(2005) and DateTime(2010)) during Interval[3, 5]
define D418: (years between DateTime(2005) and DateTime(2010)) included in Interval[3, 5]
define D419: Interval[3, 5] includes (years between DateTime(2005) and DateTime(2010))
define D420: Interval[3, 5] contains (years between DateTime(2005) and DateTime(2010))
define D421: (years between DateTime(2005) and DateTime(2010)) in Interval(3, 5)
define D422: (years between DateTime(2005) and DateTime(2010)) during Interval(3, 5)
define D423: (years between DateTime(2005) and DateTime(2010)) included in Interval(3, 5)
define D424: Interval(3, 5) includes (years between DateTime(2005) and DateTime(2010))
define D425: Interval(3, 5) contains (years between DateTime(2005) and DateTime(2010))
define D426: (years between DateTime(2005) and DateTime(2010)) in Interval[3, 5)
define D427: (years between DateTime(2005) and DateTime(2010)) during Interval[3, 5)
define D428: (years between DateTime(2005) and DateTime(2010)) included in Interval[3, 5)
define D429: Interval[3, 5) includes (years between DateTime(2005) and DateTime(2010))
define D430: Interval[3, 5) contains (years between DateTime(2005) and DateTime(2010))
define D431: (years between DateTime(2005) and DateTime(2010)) in Interval(3, 5]
define D432: (years between DateTime(2005) and DateTime(2010)) during Interval(3, 5]
define D433: (years between DateTime(2005) and DateTime(2010)) included in Interval(3, 5]
define D434: Interval(3, 5] includes (years between DateTime(2005) and DateTime(2010))
define D435: Interval(3, 5] contains (years between DateTime(2005) and DateTime(2010))
define D436: (years between DateTime(2005) and DateTime(2010)) between 3 and 5
define D437: (years between DateTime(2005) and DateTime(2010)) properly between 3 and 5
define D438: (years between DateTime(2005) and DateTime(2010)) in Interval[4, 6]
define D439: (years between DateTime(2005) and DateTime(2010)) during Interval[4, 6]
define D440: (years between DateTime(2005) and DateTime(2010)) included in Interval[4, 6]
define D441: Interval[4, 6] includes (years between DateTime(2005) and DateTime(2010))
define D442: Interval[4, 6] contains (years between DateTime(2005) and DateTime(2010))
define D443: (years between DateTime(2005) and DateTime(2010)) in Interval(4, 6)
define D444: (years between DateTime(2005) and DateTime(2010)) during Interval(4, 6)
define D445: (years between DateTime(2005) and DateTime(2010)) included in Interval(4, 6)
define D446: Interval(4, 6) includes (years between DateTime(2005) and DateTime(2010))
define D447: Interval(4, 6) contains (years between DateTime(2005) and DateTime(2010))
define D448: (years between DateTime(2005) and DateTime(2010)) in Interval[4, 6)
define D449: (years between DateTime(2005) and DateTime(2010)) during Interval[4, 6)
define D450: (years between DateTime(2005) and DateTime(2010)) included in Interval[4, 6)
define D451: Interval[4, 6) includes (years between DateTime(2005) and DateTime(2010))
define D452: Interval[4, 6) contains (years between DateTime(2005) and DateTime(2010))
define D453: (years between DateTime(2005) and DateTime(2010)) in Interval(4, 6]
define D454: (years between DateTime(2005) and DateTime(2010)) during Interval(4, 6]
define D455: (years between DateTime(2005) and DateTime(2010)) included in Interval(4, 6]
define D456: Interval(4, 6] includes (years between DateTime(2005) and DateTime(2010))
define D457: Interval(4, 6] contains (years between DateTime(2005) and DateTime(2010))
define D458: (years between DateTime(2005) and DateTime(2010)) between 4 and 6
define D459: (years between DateTime(2005) and DateTime(2010)) properly between 4 and 6
define D460: (years between DateTime(2005) and DateTime(2010)) in Interval[5, 7]
define D461: (years between DateTime(2005) and DateTime(2010)) during Interval[5, 7]
define D462: (years between DateTime(2005) and DateTime(2010)) included in Interval[5, 7]
define D463: Interval[5, 7] includes (years between DateTime(2005) and DateTime(2010))
define D464: Interval[5, 7] contains (years between DateTime(2005) and DateTime(2010))
define D465: (years between DateTime(2005) and DateTime(2010)) in Interval(5, 7)
define D466: (years between DateTime(2005) and DateTime(2010)) during Interval(5, 7)
define D467: (years between DateTime(2005) and DateTime(2010)) included in Interval(5, 7)
define D468: Interval(5, 7) includes (years between DateTime(2005) and DateTime(2010))
define D469: Interval(5, 7) contains (years between DateTime(2005) and DateTime(2010))
define D470: (years between DateTime(2005) and DateTime(2010)) in Interval[5, 7)
define D471: (years between DateTime(2005) and DateTime(2010)) during Interval[5, 7)
define D472: (years between DateTime(2005) and DateTime(2010)) included in Interval[5, 7)
define D473: Interval[5, 7) includes (years between DateTime(2005) and DateTime(2010))
define D474: Interval[5, 7) contains (years between DateTime(2005) and DateTime(2010))
define D475: (years between DateTime(2005) and DateTime(2010)) in Interval(5, 7]
define D476: (years between DateTime(2005) and DateTime(2010)) during Interval(5, 7]
define D477: (years between DateTime(2005) and DateTime(2010)) included in Interval(5, 7]
define D478: Interval(5, 7] includes (years between DateTime(2005) and DateTime(2010))
define D479: Interval(5, 7] contains (years between DateTime(2005) and DateTime(2010))
define D480: (years between DateTime(2005) and DateTime(2010)) between 5 and 7
define D481: (years between DateTime(2005) and DateTime(2010)) properly between 5 and 7
define D482: (years between DateTime(2005) and DateTime(2010)) in { 4, 5 }
define D483: (years between DateTime(2005) and DateTime(2010)) in { 8, 9 }
define D484: { 4, 8 } contains (years between DateTime(2005) and DateTime(2010))
define D485: (days between @2012-01 and @2012-02) < 0
define D486: (days between @2012-01 and @2012-02) <= 0
define D487: (days between @2012-01 and @2012-02) > 0
define D488: (days between @2012-01 and @2012-02) >= 0
define D489: (days between @2012-01 and @2012-02) = 0
define D490: (days between @2012-01 and @2012-02) before 0
define D491: (days between @2012-01 and @2012-02) after 0
define D492: (days between @2012-01 and @2012-02) same or before 0
define D493: (days between @2012-01 and @2012-02) same or after 0
define D494: (days between @2012-01 and @2012-02) same as 0
define D495: (days between @2012-01 and @2012-02) < 1
define D496: (days between @2012-01 and @2012-02) <= 1
define D497: (days between @2012-01 and @2012-02) > 1
define D498: (days between @2012-01 and @2012-02) >= 1
define D499: (days between @2012-01 and @2012-02) = 1
define D500: (days between @2012-01 and @2012-02) before 1
define D501: (days between @2012-01 and @2012-02) after 1
define D502: (days between @2012-01 and @2012-02) same or before 1
define D503: (days between @2012-01 and @2012-02) same or after 1
define D504: (days between @2012-01 and @2012-02) same as 1
define D505: (days between @2012-01 and @2012-02) < 30
define D506: (days between @2012-01 and @2012-02) <= 30
define D507: (days between @2012-01 and @2012-02) > 30
define D508: (days between @2012-01 and @2012-02) >= 30
define D509: (days between @2012-01 and @2012-02) = 30
define D510: (days between @2012-01 and @2012-02) before 30
define D511: (days between @2012-01 and @2012-02) after 30
define D512: (days between @2012-01 and @2012-02) same or before 30
define D513: (days between @2012-01 and @2012-02) same or after 30
define D514: (days between @2012-01 and @2012-02) same as 30
define D515: (days between @2012-01 and @2012-02) < 59
define D516: (days between @2012-01 and @2012-02) <= 59
define D517: (days between @2012-01 and @2012-02) > 59
define D518: (days between @2012-01 and @2012-02) >= 59
define D519: (days between @2012-01 and @2012-02) = 59
define D520: (days between @2012-01 and @2012-02) before 59
define D521: (days between @2012-01 and @2012-02) after 59
define D522: (days between @2012-01 and @2012-02) same or before 59
define D523: (days between @2012-01 and @2012-02) same or after 59
define D524: (days between @2012-01 and @2012-02) same as 59
define D525: (days between @2012-01 and @2012-02) < 60
define D526: (days between @2012-01 and @2012-02) <= 60
define D527: (days between @2012-01 and @2012-02) > 60
define D528: (days between @2012-01 and @2012-02) >= 60
define D529: (days between @2012-01 and @2012-02) = 60
define D530: (days between @2012-01 and @2012-02) before 60
define D531: (days between @2012-01 and @2012-02) after 60
define D532: (days between @2012-01 and @2012-02) same or before 60
define D533: (days between @2012-01 and @2012-02) same or after 60
define D534: (days between @2012-01 and @2012-02) same as 60
define D535: (days between @2012-01 and @2012-02) in Interval[0, 60]
define D536: (days between @2012-01 and @2012-02) during Interval[0, 60]
define D537: (days between @2012-01 and @2012-02) included in Interval[0, 60]
define D538: Interval[0, 60] includes (days between @2012-01 and @2012-02)
define D539: Interval[0, 60] contains (days between @2012-01 and @2012-02)
define D540: (days between @2012-01 and @2012-02) in Interval(0, 60)
define D541: (days between @2012-01 and @2012-02) during Interval(0, 60)
define D542: (days between @2012-01 and @2012-02) included in Interval(0, 60)
define D543: Interval(0, 60) includes (days between @2012-01 and @2012-02)
define D544: Interval(0, 60) contains (days between @2012-01 and @2012-02)
define D545: (days between @2012-01 and @2012-02) in Interval[0, 60)
define D546: (days between @2012-01 and @2012-02) during Interval[0, 60)
define D547: (days between @2012-01 and @2012-02) included in Interval[0, 60)
define D548: Interval[0, 60) includes (days between @2012-01 and @2012-02)
define D549: Interval[0, 60) contains (days between @2012-01 and @2012-02)
define D550: (days between @2012-01 and @2012-02) in Interval(0, 60]
define D551: (days between @2012-01 and @2012-02) during Interval(0, 60]
define D552: (days between @2012-01 and @2012-02) included in Interval(0, 60]
define D553: Interval(0, 60] includes (days between @2012-01 and @2012-02)
define D554: Interval(0, 60] contains (days between @2012-01 and @2012-02)
define D555: (days between @2012-01 and @2012-02) between 0 and 60
define D556: (days between @2012-01 and @2012-02) properly between 0 and 60
define D557: (days between @2012-01 and @2012-02) in Interval[1, 59]
define D558: (days between @2012-01 and @2012-02) during Interval[1, 59]
define D559: (days between @2012-01 and @2012-02) included in Interval[1, 59]
define D560: Interval[1, 59] includes (days between @2012-01 and @2012-02)
define D561: Interval[1, 59] contains (days between @2012-01 and @2012-02)
define D562: (days between @2012-01 and @2012-02) in Interval(1, 59)
define D563: (days between @2012-01 and @2012-02) during Interval(1, 59)
define D564: (days between @2012-01 and @2012-02) included in Interval(1, 59)
define D565: Interval(1, 59) includes (days between @2012-01 and @2012-02)
define D566: Interval(1, 59) contains (days between @2012-01 and @2012-02)
define D567: (days between @2012-01 and @2012-02) in Interval[1, 59)
define D568: (days between @2012-01 and @2012-02) during Interval[1, 59)
define D569: (days between @2012-01 and @2012-02) included in Interval[1, 59)
define D570: Interval[1, 59) includes (days between @2012-01 and @2012-02)
define D571: Interval[1, 59) contains (days between @2012-01 and @2012-02)
define D572: (days between @2012-01 and @2012-02) in Interval(1, 59]
define D573: (days between @2012-01 and @2012-02) during Interval(1, 59]
define D574: (days between @2012-01 and @2012-02) included in Interval(1, 59]
define D575: Interval(1, 59] includes (days between @2012-01 and @2012-02)
define D576: Interval(1, 59] contains (days between @2012-01 and @2012-02)
define D577: (days between @2012-01 and @2012-02) between 1 and 59
define D578: (days between @2012-01 and @2012-02) properly between 1 and 59
define D579: (days between @2012-01 and @2012-02) in Interval[0, 59]
define D580: (days between @2012-01 and @2012-02) during Interval[0, 59]
define D581: (days between @2012-01 and @2012-02) included in Interval[0, 59]
define D582: Interval[0, 59] includes (days between @2012-01 and @2012-02)
define D583: Interval[0, 59] contains (days between @2012-01 and @2012-02)
define D584: (days between @2012-01 and @2012-02) in Interval(0, 59)
define D585: (days between @2012-01 and @2012-02) during Interval(0, 59)
define D586: (days between @2012-01 and @2012-02) included in Interval(0, 59)
define D587: Interval(0, 59) includes (days between @2012-01 and @2012-02)
define D588: Interval(0, 59) contains (days between @2012-01 and @2012-02)
define D589: (days between @2012-01 and @2012-02) in Interval[0, 59)
define D590: (days between @2012-01 and @2012-02) during Interval[0, 59)
define D591: (days between @2012-01 and @2012-02) included in Interval[0, 59)
define D592: Interval[0, 59) includes (days between @2012-01 and @2012-02)
define D593: Interval[0, 59) contains (days between @2012-01 and @2012-02)
define D594: (days between @2012-01 and @2012-02) in Interval(0, 59]
define D595: (days between @2012-01 and @2012-02) during Interval(0, 59]
define D596: (days between @2012-01 and @2012-02) included in Interval(0, 59]
define D597: Interval(0, 59] includes (days between @2012-01 and @2012-02)
define D598: Interval(0, 59] contains (days between @2012-01 and @2012-02)
define D599: (days between @2012-01 and @2012-02) between 0 and 59
define D600: (days between @2012-01 and @2012-02) properly between 0 and 59
define D601: (days between @2012-01 and @2012-02) in Interval[1, 60]
define D602: (days between @2012-01 and @2012-02) during Interval[1, 60]
define D603: (days between @2012-01 and @2012-02) included in Interval[1, 60]
define D604: Interval[1, 60] includes (days between @2012-01 and @2012-02)
define D605: Interval[1, 60] contains (days between @2012-01 and @2012-02)
define D606: (days between @2012-01 and @2012-02) in Interval(1, 60)
define D607: (days between @2012-01 and @2012-02) during Interval(1, 60)
define D608: (days between @2012-01 and @2012-02) included in Interval(1, 60)
define D609: Interval(1, 60) includes (days between @2012-01 and @2012-02)
define D610: Interval(1, 60) contains (days between @2012-01 and @2012-02)
define D611: (days between @2012-01 and @2012-02) in Interval[1, 60)
define D612: (days between @2012-01 and @2012-02) during Interval[1, 60)
define D613: (days between @2012-01 and @2012-02) included in Interval[1, 60)
define D614: Interval[1, 60) includes (days between @2012-01 and @2012-02)
define D615: Interval[1, 60) contains (days between @2012-01 and @2012-02)
define D616: (days between @2012-01 and @2012-02) in Interval(1, 60]
define D617: (days between @2012-01 and @2012-02) during Interval(1, 60]
define D618: (days between @2012-01 and @2012-02) included in Interval(1, 60]
define D619: Interval(1, 60] includes (days between @2012-01 and @2012-02)
define D620: Interval(1, 60] contains (days between @2012-01 and @2012-02)
define D621: (days between @2012-01 and @2012-02) between 1 and 60
define D622: (days between @2012-01 and @2012-02) properly between 1 and 60
define D623: (days between @2012-01 and @2012-02) in Interval[2, 60]
define D624: (days between @2012-01 and @2012-02) during Interval[2, 60]
define D625: (days between @2012-01 and @2012-02) included in Interval[2, 60]
define D626: Interval[2, 60] includes (days between @2012-01 and @2012-02)
define D627: Interval[2, 60] contains (days between @2012-01 and @2012-02)
define D628: (days between @2012-01 and @2012-02) in Interval(2, 60)
define D629: (days between @2012-01 and @2012-02) during Interval(2, 60)
define D630: (days between @2012-01 and @2012-02) included in Interval(2, 60)
define D631: Interval(2, 60) includes (days between @2012-01 and @2012-02)
define D632: Interval(2, 60) contains (days between @2012-01 and @2012-02)
define D633: (days between @2012-01 and @2012-02) in Interval[2, 60)
define D634: (days between @2012-01 and @2012-02) during Interval[2, 60)
define D635: (days between @2012-01 and @2012-02) included in Interval[2, 60)
define D636: Interval[2, 60) includes (days between @2012-01 and @2012-02)
define D637: Interval[2, 60) contains (days between @2012-01 and @2012-02)
define D638: (days between @2012-01 and @2012-02) in Interval(2, 60]
define D639: (days between @2012-01 and @2012-02) during Interval(2, 60]
define D640: (days between @2012-01 and @2012-02) included in Interval(2, 60]
define D641: Interval(2, 60] includes (days between @2012-01 and @2012-02)
define D642: Interval(2, 60] contains (days between @2012-01 and @2012-02)
define D643: (days between @2012-01 and @2012-02) between 2 and 60
define D644: (days between @2012-01 and @2012-02) properly between 2 and 60
define D645: (days between @2012-01 and @2012-02) in { 1, 59 }
define D646: (days between @2012-01 and @2012-02) in { 62, 63 }
define D647: { 1, 62 } contains (days between @2012-01 and @2012-02)
define D648: (days between @2014-01-15 and @2014-02) < 16
define D649: (days between @2014-01-15 and @2014-02) <= 16
define D650: (days between @2014-01-15 and @2014-02) > 16
define D651: (days between @2014-01-15 and @2014-02) >= 16
define D652: (days between @2014-01-15 and @2014-02) = 16
define D653: (days between @2014-01-15 and @2014-02) before 16
define D654: (days between @2014-01-15 and @2014-02) after 16
define D655: (days between @2014-01-15 and @2014-02) same or before 16
define D656: (days between @2014-01-15 and @2014-02) same or after 16
define D657: (days between @2014-01-15 and @2014-02) same as 16
define D658: (days between @2014-01-15 and @2014-02) < 17
define D659: (days between @2014-01-15 and @2014-02) <= 17
define D660: (days between @2014-01-15 and @2014-02) > 17
define D661: (days between @2014-01-15 and @2014-02) >= 17
define D662: (days between @2014-01-15 and @2014-02) = 17
define D663: (days between @2014-01-15 and @2014-02) before 17
define D664: (days between @2014-01-15 and @2014-02) after 17
define D665: (days between @2014-01-15 and @2014-02) same or before 17
define D666: (days between @2014-01-15 and @2014-02) same or after 17
define D667: (days between @2014-01-15 and @2014-02) same as 17
define D668: (days between @2014-01-15 and @2014-02) < 30
define D669: (days between @2014-01-15 and @2014-02) <= 30
define D670: (days between @2014-01-15 and @2014-02) > 30
define D671: (days between @2014-01-15 and @2014-02) >= 30
define D672: (days between @2014-01-15 and @2014-02) = 30
define D673: (days between @2014-01-15 and @2014-02) before 30
define D674: (days between @2014-01-15 and @2014-02) after 30
define D675: (days between @2014-01-15 and @2014-02) same or before 30
define D676: (days between @2014-01-15 and @2014-02) same or after 30
define D677: (days between @2014-01-15 and @2014-02) same as 30
define D678: (days between @2014-01-15 and @2014-02) < 44
define D679: (days between @2014-01-15 and @2014-02) <= 44
define D680: (days between @2014-01-15 and @2014-02) > 44
define D681: (days between @2014-01-15 and @2014-02) >= 44
define D682: (days between @2014-01-15 and @2014-02) = 44
define D683: (days between @2014-01-15 and @2014-02) before 44
define D684: (days between @2014-01-15 and @2014-02) after 44
define D685: (days between @2014-01-15 and @2014-02) same or before 44
define D686: (days between @2014-01-15 and @2014-02) same or after 44
define D687: (days between @2014-01-15 and @2014-02) same as 44
define D688: (days between @2014-01-15 and @2014-02) < 45
define D689: (days between @2014-01-15 and @2014-02) <= 45
define D690: (days between @2014-01-15 and @2014-02) > 45
define D691: (days between @2014-01-15 and @2014-02) >= 45
define D692: (days between @2014-01-15 and @2014-02) = 45
define D693: (days between @2014-01-15 and @2014-02) before 45
define D694: (days between @2014-01-15 and @2014-02) after 45
define D695: (days between @2014-01-15 and @2014-02) same or before 45
define D696: (days between @2014-01-15 and @2014-02) same or after 45
define D697: (days between @2014-01-15 and @2014-02) same as 45
define D698: (days between @2014-01-15 and @2014-02) in Interval[16, 45]
define D699: (days between @2014-01-15 and @2014-02) during Interval[16, 45]
define D700: (days between @2014-01-15 and @2014-02) included in Interval[16, 45]
define D701: Interval[16, 45] includes (days between @2014-01-15 and @2014-02)
define D702: Interval[16, 45] contains (days between @2014-01-15 and @2014-02)
define D703: (days between @2014-01-15 and @2014-02) in Interval(16, 45)
define D704: (days between @2014-01-15 and @2014-02) during Interval(16, 45)
define D705: (days between @2014-01-15 and @2014-02) included in Interval(16, 45)
define D706: Interval(16, 45) includes (days between @2014-01-15 and @2014-02)
define D707: Interval(16, 45) contains (days between @2014-01-15 and @2014-02)
define D708: (days between @2014-01-15 and @2014-02) in Interval[16, 45)
define D709: (days between @2014-01-15 and @2014-02) during Interval[16, 45)
define D710: (days between @2014-01-15 and @2014-02) included in Interval[16, 45)
define D711: Interval[16, 45) includes (days between @2014-01-15 and @2014-02)
define D712: Interval[16, 45) contains (days between @2014-01-15 and @2014-02)
define D713: (days between @2014-01-15 and @2014-02) in Interval(16, 45]
define D714: (days between @2014-01-15 and @2014-02) during Interval(16, 45]
define D715: (days between @2014-01-15 and @2014-02) included in Interval(16, 45]
define D716: Interval(16, 45] includes (days between @2014-01-15 and @2014-02)
define D717: Interval(16, 45] contains (days between @2014-01-15 and @2014-02)
define D718: (days between @2014-01-15 and @2014-02) between 16 and 45
define D719: (days between @2014-01-15 and @2014-02) properly between 16 and 45
define D720: (days between @2014-01-15 and @2014-02) in Interval[17, 44]
define D721: (days between @2014-01-15 and @2014-02) during Interval[17, 44]
define D722: (days between @2014-01-15 and @2014-02) included in Interval[17, 44]
define D723: Interval[17, 44] includes (days between @2014-01-15 and @2014-02)
define D724: Interval[17, 44] contains (days between @2014-01-15 and @2014-02)
define D725: (days between @2014-01-15 and @2014-02) in Interval(17, 44)
define D726: (days between @2014-01-15 and @2014-02) during Interval(17, 44)
define D727: (days between @2014-01-15 and @2014-02) included in Interval(17, 44)
define D728: Interval(17, 44) includes (days between @2014-01-15 and @2014-02)
define D729: Interval(17, 44) contains (days between @2014-01-15 and @2014-02)
define D730: (days between @2014-01-15 and @2014-02) in Interval[17, 44)
define D731: (days between @2014-01-15 and @2014-02) during Interval[17, 44)
define D732: (days between @2014-01-15 and @2014-02) included in Interval[17, 44)
define D733: Interval[17, 44) includes (days between @2014-01-15 and @2014-02)
define D734: Interval[17, 44) contains (days between @2014-01-15 and @2014-02)
define D735: (days between @2014-01-15 and @2014-02) in Interval(17, 44]
define D736: (days between @2014-01-15 and @2014-02) during Interval(17, 44]
define D737: (days between @2014-01-15 and @2014-02) included in Interval(17, 44]
define D738: Interval(17, 44] includes (days between @2014-01-15 and @2014-02)
define D739: Interval(17, 44] contains (days between @2014-01-15 and @2014-02)
define D740: (days between @2014-01-15 and @2014-02) between 17 and 44
define D741: (days between @2014-01-15 and @2014-02) properly between 17 and 44
define D742: (days between @2014-01-15 and @2014-02) in Interval[16, 44]
define D743: (days between @2014-01-15 and @2014-02) during Interval[16, 44]
define D744: (days between @2014-01-15 and @2014-02) included in Interval[16, 44]
define D745: Interval[16, 44] includes (days between @2014-01-15 and @2014-02)
define D746: Interval[16, 44] contains (days between @2014-01-15 and @2014-02)
define D747: (days between @2014-01-15 and @2014-02) in Interval(16, 44)
define D748: (days between @2014-01-15 and @2014-02) during Interval(16, 44)
define D749: (days between @2014-01-15 and @2014-02) included in Interval(16, 44)
define D750: Interval(16, 44) includes (days between @2014-01-15 and @2014-02)
define D751: Interval(16, 44) contains (days between @2014-01-15 and @2014-02)
define D752: (days between @2014-01-15 and @2014-02) in Interval[16, 44)
define D753: (days between @2014-01-15 and @2014-02) during Interval[16, 44)
define D754: (days between @2014-01-15 and @2014-02) included in Interval[16, 44)
define D755: Interval[16, 44) includes (days between @2014-01-15 and @2014-02)
define D756: Interval[16, 44) contains (days between @2014-01-15 and @2014-02)
define D757: (days between @2014-01-15 and @2014-02) in Interval(16, 44]
define D758: (days between @2014-01-15 and @2014-02) during Interval(16, 44]
define D759: (days between @2014-01-15 and @2014-02) included in Interval(16, 44]
define D760: Interval(16, 44] includes (days between @2014-01-15 and @2014-02)
define D761: Interval(16, 44] contains (days between @2014-01-15 and @2014-02)
define D762: (days between @2014-01-15 and @2014-02) between 16 and 44
define D763: (days between @2014-01-15 and @2014-02) properly between 16 and 44
define D764: (days between @2014-01-15 and @2014-02) in Interval[17, 45]
define D765: (days between @2014-01-15 and @2014-02) during Interval[17, 45]
define D766: (days between @2014-01-15 and @2014-02) included in Interval[17, 45]
define D767: Interval[17, 45] includes (days between @2014-01-15 and @2014-02)
define D768: Interval[17, 45] contains (days between @2014-01-15 and @2014-02)
define D769: (days between @2014-01-15 and @2014-02) in Interval(17, 45)
define D770: (days between @2014-01-15 and @2014-02) during Interval(17, 45)
define D771: (days between @2014-01-15 and @2014-02) included in Interval(17, 45)
define D772: Interval(17, 45) includes (days between @2014-01-15 and @2014-02)
define D773: Interval(17, 45) contains (days between @2014-01-15 and @2014-02)
define D774: (days between @2014-01-15 and @2014-02) in Interval[17, 45)
define D775: (days between @2014-01-15 and @2014-02) during Interval[17, 45)
define D776: (days between @2014-01-15 and @2014-02) included in Interval[17, 45)
define D777: Interval[17, 45) includes (days between @2014-01-15 and @2014-02)
define D778: Interval[17, 45) contains (days between @2014-01-15 and @2014-02)
define D779: (days between @2014-01-15 and @2014-02) in Interval(17, 45]
define D780: (days between @2014-01-15 and @2014-02) during Interval(17, 45]
define D781: (days between @2014-01-15 and @2014-02) included in Interval(17, 45]
define D782: Interval(17, 45] includes (days between @2014-01-15 and @2014-02)
define D783: Interval(17, 45] contains (days between @2014-01-15 and @2014-02)
define D784: (days between @2014-01-15 and @2014-02) between 17 and 45
define D785: (days between @2014-01-15 and @2014-02) properly between 17 and 45
define D786: (days between @2014-01-15 and @2014-02) in Interval[18, 45]
define D787: (days between @2014-01-15 and @2014-02) during Interval[18, 45]
define D788: (days between @2014-01-15 and @2014-02) included in Interval[18, 45]
define D789: Interval[18, 45] includes (days between @2014-01-15 and @2014-02)
define D790: Interval[18, 45] contains (days between @2014-01-15 and @2014-02)
define D791: (days between @2014-01-15 and @2014-02) in Interval(18, 45)
define D792: (days between @2014-01-15 and @2014-02) during Interval(18, 45)
define D793: (days between @2014-01-15 and @2014-02) included in Interval(18, 45)
define D794: Interval(18, 45) includes (days between @2014-01-15 and @2014-02)
define D795: Interval(18, 45) contains (days between @2014-01-15 and @2014-02)
define D796: (days between @2014-01-15 and @2014-02) in Interval[18, 45)
define D797: (days between @2014-01-15 and @2014-02) during Interval[18, 45)
define D798: (days between @2014-01-15 and @2014-02) included in Interval[18, 45)
define D799: Interval[18, 45) includes (days between @2014-01-15 and @2014-02)
define D800: Interval[18, 45) contains (days between @2014-01-15 and @2014-02)
define D801: (days between @2014-01-15 and @2014-02) in Interval(18, 45]
define D802: (days between @2014-01-15 and @2014-02) during Interval(18, 45]
define D803: (days between @2014-01-15 and @2014-02) included in Interval(18, 45]
define D804: Interval(18, 45] includes (days between @2014-01-15 and @2014-02)
define D805: Interval(18, 45] contains (days between @2014-01-15 and @2014-02)
define D806: (days between @2014-01-15 and @2014-02) between 18 and 45
define D807: (days between @2014-01-15 and @2014-02) properly between 18 and 45
define D808: (days between @2014-01-15 and @2014-02) in { 17, 44 }
define D809: (days between @2014-01-15 and @2014-02) in { 47, 48 }
define D810: { 17, 47 } contains (days between @2014-01-15 and @2014-02)
define D811: (days between @2017-08-07T17:00 and @2017-08-14T) < 5
define D812: (days between @2017-08-07T17:00 and @2017-08-14T) <= 5
define D813: (days between @2017-08-07T17:00 and @2017-08-14T) > 5
define D814: (days between @2017-08-07T17:00 and @2017-08-14T) >= 5
define D815: (days between @2017-08-07T17:00 and @2017-08-14T) = 5
define D816: (days between @2017-08-07T17:00 and @2017-08-14T) before 5
define D817: (days between @2017-08-07T17:00 and @2017-08-14T) after 5
define D818: (days between @2017-08-07T17:00 and @2017-08-14T) same or before 5
define D819: (days between @2017-08-07T17:00 and @2017-08-14T) same or after 5
define D820: (days between @2017-08-07T17:00 and @2017-08-14T) same as 5
define D821: (days between @2017-08-07T17:00 and @2017-08-14T) < 6
define D822: (days between @2017-08-07T17:00 and @2017-08-14T) <= 6
define D823: (days between @2017-08-07T17:00 and @2017-08-14T) > 6
define D824: (days between @2017-08-07T17:00 and @2017-08-14T) >= 6
define D825: (days between @2017-08-07T17:00 and @2017-08-14T) = 6
define D826: (days between @2017-08-07T17:00 and @2017-08-14T) before 6
define D827: (days between @2017-08-07T17:00 and @2017-08-14T) after 6
define D828: (days between @2017-08-07T17:00 and @2017-08-14T) same or before 6
define D829: (days between @2017-08-07T17:00 and @2017-08-14T) same or after 6
define D830: (days between @2017-08-07T17:00 and @2017-08-14T) same as 6
define D831: (days between @2017-08-07T17:00 and @2017-08-14T) < 7
define D832: (days between @2017-08-07T17:00 and @2017-08-14T) <= 7
define D833: (days between @2017-08-07T17:00 and @2017-08-14T) > 7
define D834: (days between @2017-08-07T17:00 and @2017-08-14T) >= 7
define D835: (days between @2017-08-07T17:00 and @2017-08-14T) = 7
define D836: (days between @2017-08-07T17:00 and @2017-08-14T) before 7
define D837: (days between @2017-08-07T17:00 and @2017-08-14T) after 7
define D838: (days between @2017-08-07T17:00 and @2017-08-14T) same or before 7
define D839: (days between @2017-08-07T17:00 and @2017-08-14T) same or after 7
define D840: (days between @2017-08-07T17:00 and @2017-08-14T) same as 7
define D841: (days between @2017-08-07T17:00 and @2017-08-14T) < 8
define D842: (days between @2017-08-07T17:00 and @2017-08-14T) <= 8
define D843: (days between @2017-08-07T17:00 and @2017-08-14T) > 8
define D844: (days between @2017-08-07T17:00 and @2017-08-14T) >= 8
define D845: (days between @2017-08-07T17:00 and @2017-08-14T) = 8
define D846: (days between @2017-08-07T17:00 and @2017-08-14T) before 8
define D847: (days between @2017-08-07T17:00 and @2017-08-14T) after 8
define D848: (days between @2017-08-07T17:00 and @2017-08-14T) same or before 8
define D849: (days between @2017-08-07T17:00 and @2017-08-14T) same or after 8
define D850: (days between @2017-08-07T17:00 and @2017-08-14T) same as 8
define D851: (days between @2017-08-07T17:00 and @2017-08-14T) in Interval[5, 8]
define D852: (days between @2017-08-07T17:00 and @2017-08-14T) during Interval[5, 8]
define D853: (days between @2017-08-07T17:00 and @2017-08-14T) included in Interval[5, 8]
define D854: Interval[5, 8] includes (days between @2017-08-07T17:00 and @2017-08-14T)
define D855: Interval[5, 8] contains (days between @2017-08-07T17:00 and @2017-08-14T)
define D856: (days between @2017-08-07T17:00 and @2017-08-14T) in Interval(5, 8)
define D857: (days between @2017-08-07T17:00 and @2017-08-14T) during Interval(5, 8)
define D858: (days between @2017-08-07T17:00 and @2017-08-14T) included in Interval(5, 8)
define D859: Interval(5, 8) includes (days between @2017-08-07T17:00 and @2017-08-14T)
define D860: Interval(5, 8) contains (days between @2017-08-07T17:00 and @2017-08-14T)
define D861: (days between @2017-08-07T17:00 and @2017-08-14T) in Interval[5, 8)
define D862: (days between @2017-08-07T17:00 and @2017-08-14T) during Interval[5, 8)
define D863: (days between @2017-08-07T17:00 and @2017-08-14T) included in Interval[5, 8)
define D864: Interval[5, 8) includes (days between @2017-08-07T17:00 and @2017-08-14T)
define D865: Interval[5, 8) contains (days between @2017-08-07T17:00 and @2017-08-14T)
define D866: (days between @2017-08-07T17:00 and @2017-08-14T) in Interval(5, 8]
define D867: (days between @2017-08-07T17:00 and @2017-08-14T) during Interval(5, 8]
define D868: (days between @2017-08-07T17:00 and @2017-08-14T) included in Interval(5, 8]
define D869: Interval(5, 8] includes (days between @2017-08-07T17:00 and @2017-08-14T)
define D870: Interval(5, 8] contains (days between @2017-08-07T17:00 and @2017-08-14T)
define D871: (days between @2017-08-07T17:00 and @2017-08-14T) between 5 and 8
define D872: (days between @2017-08-07T17:00 and @2017-08-14T) properly between 5 and 8
define D873: (days between @2017-08-07T17:00 and @2017-08-14T) in Interval[6, 8]
define D874: (days between @2017-08-07T17:00 and @2017-08-14T) during Interval[6, 8]
define D875: (days between @2017-08-07T17:00 and @2017-08-14T) included in Interval[6, 8]
define D876: Interval[6, 8] includes (days between @2017-08-07T17:00 and @2017-08-14T)
define D877: Interval[6, 8] contains (days between @2017-08-07T17:00 and @2017-08-14T)
define D878: (days between @2017-08-07T17:00 and @2017-08-14T) in Interval(6, 8)
define D879: (days between @2017-08-07T17:00 and @2017-08-14T) during Interval(6, 8)
define D880: (days between @2017-08-07T17:00 and @2017-08-14T) included in Interval(6, 8)
define D881: Interval(6, 8) includes (days between @2017-08-07T17:00 and @2017-08-14T)
define D882: Interval(6, 8) contains (days between @2017-08-07T17:00 and @2017-08-14T)
define D883: (days between @2017-08-07T17:00 and @2017-08-14T) in Interval[6, 8)
define D884: (days between @2017-08-07T17:00 and @2017-08-14T) during Interval[6, 8)
define D885: (days between @2017-08-07T17:00 and @2017-08-14T) included in Interval[6, 8)
define D886: Interval[6, 8) includes (days between @2017-08-07T17:00 and @2017-08-14T)
define D887: Interval[6, 8) contains (days between @2017-08-07T17:00 and @2017-08-14T)
define D888: (days between @2017-08-07T17:00 and @2017-08-14T) in Interval(6, 8]
define D889: (days between @2017-08-07T17:00 and @2017-08-14T) during Interval(6, 8]
define D890: (days between @2017-08-07T17:00 and @2017-08-14T) included in Interval(6, 8]
define D891: Interval(6, 8] includes (days between @2017-08-07T17:00 and @2017-08-14T)
define D892: Interval(6, 8] contains (days between @2017-08-07T17:00 and @2017-08-14T)
define D893: (days between @2017-08-07T17:00 and @2017-08-14T) between 6 and 8
define D894: (days between @2017-08-07T17:00 and @2017-08-14T) properly between 6 and 8
define D895: (days between @2017-08-07T17:00 and @2017-08-14T) in Interval[5, 7]
define D896: (days between @2017-08-07T17:00 and @2017-08-14T) during Interval[5, 7]
define D897: (days between @2017-08-07T17:00 and @2017-08-14T) included in Interval[5, 7]
define D898: Interval[5, 7] includes (days between @2017-08-07T17:00 and @2017-08-14T)
define D899: Interval[5, 7] contains (days between @2017-08-07T17:00 and @2017-08-14T)
define D900: (days between @2017-08-07T17:00 and @2017-08-14T) in Interval(5, 7)
define D901: (days between @2017-08-07T17:00 and @2017-08-14T) during Interval(5, 7)
define D902: (days between @2017-08-07T17:00 and @2017-08-14T) included in Interval(5, 7)
define D903: Interval(5, 7) includes (days between @2017-08-07T17:00 and @2017-08-14T)
define D904: Interval(5, 7) contains (days between @2017-08-07T17:00 and @2017-08-14T)
define D905: (days between @2017-08-07T17:00 and @2017-08-14T) in Interval[5, 7)
define D906: (days between @2017-08-07T17:00 and @2017-08-14T) during Interval[5, 7)
define D907: (days between @2017-08-07T17:00 and @2017-08-14T) included in Interval[5, 7)
define D908: Interval[5, 7) includes (days between @2017-08-07T17:00 and @2017-08-14T)
define D909: Interval[5, 7) contains (days between @2017-08-07T17:00 and @2017-08-14T)
define D910: (days between @2017-08-07T17:00 and @2017-08-14T) in Interval(5, 7]
define D911: (days between @2017-08-07T17:00 and @2017-08-14T) during Interval(5, 7]
define D912: (days between @2017-08-07T17:00 and @2017-08-14T) included in Interval(5, 7]
define D913: Interval(5, 7] includes (days between @2017-08-07T17:00 and @2017-08-14T)
define D914: Interval(5, 7] contains (days between @2017-08-07T17:00 and @2017-08-14T)
define D915: (days between @2017-08-07T17:00 and @2017-08-14T) between 5 and 7
define D916: (days between @2017-08-07T17:00 and @2017-08-14T) properly between 5 and 7
define D917: (days between @2017-08-07T17:00 and @2017-08-14T) in Interval[6, 8]
define D918: (days between @2017-08-07T17:00 and @2017-08-14T) during Interval[6, 8]
define D919: (days between @2017-08-07T17:00 and @2017-08-14T) included in Interval[6, 8]
define D920: Interval[6, 8] includes (days between @2017-08-07T17:00 and @2017-08-14T)
define D921: Interval[6, 8] contains (days between @2017-08-07T17:00 and @2017-08-14T)
define D922: (days between @2017-08-07T17:00 and @2017-08-14T) in Interval(6, 8)
define D923: (days between @2017-08-07T17:00 and @2017-08-14T) during Interval(6, 8)
define D924: (days between @2017-08-07T17:00 and @2017-08-14T) included in Interval(6, 8)
define D925: Interval(6, 8) includes (days between @2017-08-07T17:00 and @2017-08-14T)
define D926: Interval(6, 8) contains (days between @2017-08-07T17:00 and @2017-08-14T)
define D927: (days between @2017-08-07T17:00 and @2017-08-14T) in Interval[6, 8)
define D928: (days between @2017-08-07T17:00 and @2017-08-14T) during Interval[6, 8)
define D929: (days between @2017-08-07T17:00 and @2017-08-14T) included in Interval[6, 8)
define D930: Interval[6, 8) includes (days between @2017-08-07T17:00 and @2017-08-14T)
define D931: Interval[6, 8) contains (days between @2017-08-07T17:00 and @2017-08-14T)
define D932: (days between @2017-08-07T17:00 and @2017-08-14T) in Interval(6, 8]
define D933: (days between @2017-08-07T17:00 and @2017-08-14T) during Interval(6, 8]
define D934: (days between @2017-08-07T17:00 and @2017-08-14T) included in Interval(6, 8]
define D935: Interval(6, 8] includes (days between @2017-08-07T17:00 and @2017-08-14T)
define D936: Interval(6, 8] contains (days between @2017-08-07T17:00 and @2017-08-14T)
define D937: (days between @2017-08-07T17:00 and @2017-08-14T) between 6 and 8
define D938: (days between @2017-08-07T17:00 and @2017-08-14T) properly between 6 and 8
define D939: (days between @2017-08-07T17:00 and @2017-08-14T) in Interval[7, 9]
define D940: (days between @2017-08-07T17:00 and @2017-08-14T) during Interval[7, 9]
define D941: (days between @2017-08-07T17:00 and @2017-08-14T) included in Interval[7, 9]
define D942: Interval[7, 9] includes (days between @2017-08-07T17:00 and @2017-08-14T)
define D943: Interval[7, 9] contains (days between @2017-08-07T17:00 and @2017-08-14T)
define D944: (days between @2017-08-07T17:00 and @2017-08-14T) in Interval(7, 9)
define D945: (days between @2017-08-07T17:00 and @2017-08-14T) during Interval(7, 9)
define D946: (days between @2017-08-07T17:00 and @2017-08-14T) included in Interval(7, 9)
define D947: Interval(7, 9) includes (days between @2017-08-07T17:00 and @2017-08-14T)
define D948: Interval(7, 9) contains (days between @2017-08-07T17:00 and @2017-08-14T)
define D949: (days between @2017-08-07T17:00 and @2017-08-14T) in Interval[7, 9)
define D950: (days between @2017-08-07T17:00 and @2017-08-14T) during Interval[7, 9)
define D951: (days between @2017-08-07T17:00 and @2017-08-14T) included in Interval[7, 9)
define D952: Interval[7, 9) includes (days between @2017-08-07T17:00 and @2017-08-14T)
define D953: Interval[7, 9) contains (days between @2017-08-07T17:00 and @2017-08-14T)
define D954: (days between @2017-08-07T17:00 and @2017-08-14T) in Interval(7, 9]
define D955: (days between @2017-08-07T17:00 and @2017-08-14T) during Interval(7, 9]
define D956: (days between @2017-08-07T17:00 and @2017-08-14T) included in Interval(7, 9]
define D957: Interval(7, 9] includes (days between @2017-08-07T17:00 and @2017-08-14T)
define D958: Interval(7, 9] contains (days between @2017-08-07T17:00 and @2017-08-14T)
define D959: (days between @2017-08-07T17:00 and @2017-08-14T) between 7 and 9
define D960: (days between @2017-08-07T17:00 and @2017-08-14T) properly between 7 and 9
define D961: (days between @2017-08-07T17:00 and @2017-08-14T) in { 6, 7 }
define D962: (days between @2017-08-07T17:00 and @2017-08-14T) in { 10, 11 }
define D963: { 6, 10 } contains (days between @2017-08-07T17:00 and @2017-08-14T)
define D964: (months between DateTime(2005) and DateTime(2006, 7)) < 5
define D965: (months between DateTime(2005) and DateTime(2006, 7)) <= 5
define D966: (months between DateTime(2005) and DateTime(2006, 7)) > 5
define D967: (months between DateTime(2005) and DateTime(2006, 7)) >= 5
define D968: (months between DateTime(2005) and DateTime(2006, 7)) = 5
define D969: (months between DateTime(2005) and DateTime(2006, 7)) before 5
define D970: (months between DateTime(2005) and DateTime(2006, 7)) after 5
define D971: (months between DateTime(2005) and DateTime(2006, 7)) same or before 5
define D972: (months between DateTime(2005) and DateTime(2006, 7)) same or after 5
define D973: (months between DateTime(2005) and DateTime(2006, 7)) same as 5
define D974: (months between DateTime(2005) and DateTime(2006, 7)) < 6
define D975: (months between DateTime(2005) and DateTime(2006, 7)) <= 6
define D976: (months between DateTime(2005) and DateTime(2006, 7)) > 6
define D977: (months between DateTime(2005) and DateTime(2006, 7)) >= 6
define D978: (months between DateTime(2005) and DateTime(2006, 7)) = 6
define D979: (months between DateTime(2005) and DateTime(2006, 7)) before 6
define D980: (months between DateTime(2005) and DateTime(2006, 7)) after 6
define D981: (months between DateTime(2005) and DateTime(2006, 7)) same or before 6
define D982: (months between DateTime(2005) and DateTime(2006, 7)) same or after 6
define D983: (months between DateTime(2005) and DateTime(2006, 7)) same as 6
define D984: (months between DateTime(2005) and DateTime(2006, 7)) < 12
define D985: (months between DateTime(2005) and DateTime(2006, 7)) <= 12
define D986: (months between DateTime(2005) and DateTime(2006, 7)) > 12
define D987: (months between DateTime(2005) and DateTime(2006, 7)) >= 12
define D988: (months between DateTime(2005) and DateTime(2006, 7)) = 12
define D989: (months between DateTime(2005) and DateTime(2006, 7)) before 12
define D990: (months between DateTime(2005) and DateTime(2006, 7)) after 12
define D991: (months between DateTime(2005) and DateTime(2006, 7)) same or before 12
define D992: (months between DateTime(2005) and DateTime(2006, 7)) same or after 12
define D993: (months between DateTime(2005) and DateTime(2006, 7)) same as 12
define D994: (months between DateTime(2005) and DateTime(2006, 7)) < 18
define D995: (months between DateTime(2005) and DateTime(2006, 7)) <= 18
define D996: (months between DateTime(2005) and DateTime(2006, 7)) > 18
define D997: (months between DateTime(2005) and DateTime(2006, 7)) >= 18
define D998: (months between DateTime(2005) and DateTime(2006, 7)) = 18
define D999: (months between DateTime(2005) and DateTime(2006, 7)) before 18
define D1000: (months between DateTime(2005) and DateTime(2006, 7)) after 18
define D1001: (months between DateTime(2005) and DateTime(2006, 7)) same or before 18
define D1002: (months between DateTime(2005) and DateTime(2006, 7)) same or after 18
define D1003: (months between DateTime(2005) and DateTime(2006, 7)) same as 18
define D1004: (months between DateTime(2005) and DateTime(2006, 7)) < 19
define D1005: (months between DateTime(2005) and DateTime(2006, 7)) <= 19
define D1006: (months between DateTime(2005) and DateTime(2006, 7)) > 19
define D1007: (months between DateTime(2005) and DateTime(2006, 7)) >= 19
define D1008: (months between DateTime(2005) and DateTime(2006, 7)) = 19
define D1009: (months between DateTime(2005) and DateTime(2006, 7)) before 19
define D1010: (months between DateTime(2005) and DateTime(2006, 7)) after 19
define D1011: (months between DateTime(2005) and DateTime(2006, 7)) same or before 19
define D1012: (months between DateTime(2005) and DateTime(2006, 7)) same or after 19
define D1013: (months between DateTime(2005) and DateTime(2006, 7)) same as 19
define D1014: (months between DateTime(2005) and DateTime(2006, 7)) in Interval[5, 19]
define D1015: (months between DateTime(2005) and DateTime(2006, 7)) during Interval[5, 19]
define D1016: (months between DateTime(2005) and DateTime(2006, 7)) included in Interval[5, 19]
define D1017: Interval[5, 19] includes (months between DateTime(2005) and DateTime(2006, 7))
define D1018: Interval[5, 19] contains (months between DateTime(2005) and DateTime(2006, 7))
define D1019: (months between DateTime(2005) and DateTime(2006, 7)) in Interval(5, 19)
define D1020: (months between DateTime(2005) and DateTime(2006, 7)) during Interval(5, 19)
define D1021: (months between DateTime(2005) and DateTime(2006, 7)) included in Interval(5, 19)
define D1022: Interval(5, 19) includes (months between DateTime(2005) and DateTime(2006, 7))
define D1023: Interval(5, 19) contains (months between DateTime(2005) and DateTime(2006, 7))
define D1024: (months between DateTime(2005) and DateTime(2006, 7)) in Interval[5, 19)
define D1025: (months between DateTime(2005) and DateTime(2006, 7)) during Interval[5, 19)
define D1026: (months between DateTime(2005) and DateTime(2006, 7)) included in Interval[5, 19)
define D1027: Interval[5, 19) includes (months between DateTime(2005) and DateTime(2006, 7))
define D1028: Interval[5, 19) contains (months between DateTime(2005) and DateTime(2006, 7))
define D1029: (months between DateTime(2005) and DateTime(2006, 7)) in Interval(5, 19]
define D1030: (months between DateTime(2005) and DateTime(2006, 7)) during Interval(5, 19]
define D1031: (months between DateTime(2005) and DateTime(2006, 7)) included in Interval(5, 19]
define D1032: Interval(5, 19] includes (months between DateTime(2005) and DateTime(2006, 7))
define D1033: Interval(5, 19] contains (months between DateTime(2005) and DateTime(2006, 7))
define D1034: (months between DateTime(2005) and DateTime(2006, 7)) between 5 and 19
define D1035: (months between DateTime(2005) and DateTime(2006, 7)) properly between 5 and 19
define D1036: (months between DateTime(2005) and DateTime(2006, 7)) in Interval[6, 18]
define D1037: (months between DateTime(2005) and DateTime(2006, 7)) during Interval[6, 18]
define D1038: (months between DateTime(2005) and DateTime(2006, 7)) included in Interval[6, 18]
define D1039: Interval[6, 18] includes (months between DateTime(2005) and DateTime(2006, 7))
define D1040: Interval[6, 18] contains (months between DateTime(2005) and DateTime(2006, 7))
define D1041: (months between DateTime(2005) and DateTime(2006, 7)) in Interval(6, 18)
define D1042: (months between DateTime(2005) and DateTime(2006, 7)) during Interval(6, 18)
define D1043: (months between DateTime(2005) and DateTime(2006, 7)) included in Interval(6, 18)
define D1044: Interval(6, 18) includes (months between DateTime(2005) and DateTime(2006, 7))
define D1045: Interval(6, 18) contains (months between DateTime(2005) and DateTime(2006, 7))
define D1046: (months between DateTime(2005) and DateTime(2006, 7)) in Interval[6, 18)
define D1047: (months between DateTime(2005) and DateTime(2006, 7)) during Interval[6, 18)
define D1048: (months between DateTime(2005) and DateTime(2006, 7)) included in Interval[6, 18)
define D1049: Interval[6, 18) includes (months between DateTime(2005) and DateTime(2006, 7))
define D1050: Interval[6, 18) contains (months between DateTime(2005) and DateTime(2006, 7))
define D1051: (months between DateTime(2005) and DateTime(2006, 7)) in Interval(6, 18]
define D1052: (months between DateTime(2005) and DateTime(2006, 7)) during Interval(6, 18]
define D1053: (months between DateTime(2005) and DateTime(2006, 7)) included in Interval(6, 18]
define D1054: Interval(6, 18] includes (months between DateTime(2005) and DateTime(2006, 7))
define D1055: Interval(6, 18] contains (months between DateTime(2005) and DateTime(2006, 7))
define D1056: (months between DateTime(2005) and DateTime(2006, 7)) between 6 and 18
define D1057: (months between DateTime(2005) and DateTime(2006, 7)) properly between 6 and 18
define D1058: (months between DateTime(2005) and DateTime(2006, 7)) in Interval[5, 18]
define D1059: (months between DateTime(2005) and DateTime(2006, 7)) during Interval[5, 18]
define D1060: (months between DateTime(2005) and DateTime(2006, 7)) included in Interval[5, 18]
define D1061: Interval[5, 18] includes (months between DateTime(2005) and DateTime(2006, 7))
define D1062: Interval[5, 18] contains (months between DateTime(2005) and DateTime(2006, 7))
define D1063: (months between DateTime(2005) and DateTime(2006, 7)) in Interval(5, 18)
define D1064: (months between DateTime(2005) and DateTime(2006, 7)) during Interval(5, 18)
define D1065: (months between DateTime(2005) and DateTime(2006, 7)) included in Interval(5, 18)
define D1066: Interval(5, 18) includes (months between DateTime(2005) and DateTime(2006, 7))
define D1067: Interval(5, 18) contains (months between DateTime(2005) and DateTime(2006, 7))
define D1068: (months between DateTime(2005) and DateTime(2006, 7)) in Interval[5, 18)
define D1069: (months between DateTime(2005) and DateTime(2006, 7)) during Interval[5, 18)
define D1070: (months between DateTime(2005) and DateTime(2006, 7)) included in Interval[5, 18)
define D1071: Interval[5, 18) includes (months between DateTime(2005) and DateTime(2006, 7))
define D1072: Interval[5, 18) contains (months between DateTime(2005) and DateTime(2006, 7))
define D1073: (months between DateTime(2005) and DateTime(2006, 7)) in Interval(5, 18]
define D1074: (months between DateTime(2005) and DateTime(2006, 7)) during Interval(5, 18]
define D1075: (months between DateTime(2005) and DateTime(2006, 7)) included in Interval(5, 18]
define D1076: Interval(5, 18] includes (months between DateTime(2005) and DateTime(2006, 7))
define D1077: Interval(5, 18] contains (months between DateTime(2005) and DateTime(2006, 7))
define D1078: (months between DateTime(2005) and DateTime(2006, 7)) between 5 and 18
define D1079: (months between DateTime(2005) and DateTime(2006, 7)) properly between 5 and 18
define D1080: (months between DateTime(2005) and DateTime(2006, 7)) in Interval[6, 19]
define D1081: (months between DateTime(2005) and DateTime(2006, 7)) during Interval[6, 19]
define D1082: (months between DateTime(2005) and DateTime(2006, 7)) included in Interval[6, 19]
define D1083: Interval[6, 19] includes (months between DateTime(2005) and DateTime(2006, 7))
define D1084: Interval[6, 19] contains (months between DateTime(2005) and DateTime(2006, 7))
define D1085: (months between DateTime(2005) and DateTime(2006, 7)) in Interval(6, 19)
define D1086: (months between DateTime(2005) and DateTime(2006, 7)) during Interval(6, 19)
define D1087: (months between DateTime(2005) and DateTime(2006, 7)) included in Interval(6, 19)
define D1088: Interval(6, 19) includes (months between DateTime(2005) and DateTime(2006, 7))
define D1089: Interval(6, 19) contains (months between DateTime(2005) and DateTime(2006, 7))
define D1090: (months between DateTime(2005) and DateTime(2006, 7)) in Interval[6, 19)
define D1091: (months between DateTime(2005) and DateTime(2006, 7)) during Interval[6, 19)
define D1092: (months between DateTime(2005) and DateTime(2006, 7)) included in Interval[6, 19)
define D1093: Interval[6, 19) includes (months between DateTime(2005) and DateTime(2006, 7))
define D1094: Interval[6, 19) contains (months between DateTime(2005) and DateTime(2006, 7))
define D1095: (months between DateTime(2005) and DateTime(2006, 7)) in Interval(6, 19]
define D1096: (months between DateTime(2005) and DateTime(2006, 7)) during Interval(6, 19]
define D1097: (months between DateTime(2005) and DateTime(2006, 7)) included in Interval(6, 19]
define D1098: Interval(6, 19] includes (months between DateTime(2005) and DateTime(2006, 7))
define D1099: Interval(6, 19] contains (months between DateTime(2005) and DateTime(2006, 7))
define D1100: (months between DateTime(2005) and DateTime(2006, 7)) between 6 and 19
define D1101: (months between DateTime(2005) and DateTime(2006, 7)) properly between 6 and 19
define D1102: (months between DateTime(2005) and DateTime(2006, 7)) in Interval[7, 19]
define D1103: (months between DateTime(2005) and DateTime(2006, 7)) during Interval[7, 19]
define D1104: (months between DateTime(2005) and DateTime(2006, 7)) included in Interval[7, 19]
define D1105: Interval[7, 19] includes (months between DateTime(2005) and DateTime(2006, 7))
define D1106: Interval[7, 19] contains (months between DateTime(2005) and DateTime(2006, 7))
define D1107: (months between DateTime(2005) and DateTime(2006, 7)) in Interval(7, 19)
define D1108: (months between DateTime(2005) and DateTime(2006, 7)) during Interval(7, 19)
define D1109: (months between DateTime(2005) and DateTime(2006, 7)) included in Interval(7, 19)
define D1110: Interval(7, 19) includes (months between DateTime(2005) and DateTime(2006, 7))
define D1111: Interval(7, 19) contains (months between DateTime(2005) and DateTime(2006, 7))
define D1112: (months between DateTime(2005) and DateTime(2006, 7)) in Interval[7, 19)
define D1113: (months between DateTime(2005) and DateTime(2006, 7)) during Interval[7, 19)
define D1114: (months between DateTime(2005) and DateTime(2006, 7)) included in Interval[7, 19)
define D1115: Interval[7, 19) includes (months between DateTime(2005) and DateTime(2006, 7))
define D1116: Interval[7, 19) contains (months between DateTime(2005) and DateTime(2006, 7))
define D1117: (months between DateTime(2005) and DateTime(2006, 7)) in Interval(7, 19]
define D1118: (months between DateTime(2005) and DateTime(2006, 7)) during Interval(7, 19]
define D1119: (months between DateTime(2005) and DateTime(2006, 7)) included in Interval(7, 19]
define D1120: Interval(7, 19] includes (months between DateTime(2005) and DateTime(2006, 7))
define D1121: Interval(7, 19] contains (months between DateTime(2005) and DateTime(2006, 7))
define D1122: (months between DateTime(2005) and DateTime(2006, 7)) between 7 and 19
define D1123: (months between DateTime(2005) and DateTime(2006, 7)) properly between 7 and 19
define D1124: (months between DateTime(2005) and DateTime(2006, 7)) in { 6, 18 }
define D1125: (months between DateTime(2005) and DateTime(2006, 7)) in { 21, 22 }
define D1126: { 6, 21 } contains (months between DateTime(2005) and DateTime(2006, 7))
define D1127: (years between @2012 and @2014) < (years between @2012 and @2014)
define D1128: (years between @2012 and @2014) <= (years between @2012 and @2014)
define D1129: (years between @2012 and @2014) > (years between @2012 and @2014)
define D1130: (years between @2012 and @2014) >= (years between @2012 and @2014)
define D1131: (years between @2012 and @2014) = (years between @2012 and @2014)
define D1132: (years between @2012 and @2014) < (days between @2012-01 and @2012-02)
define D1133: (years between @2012 and @2014) <= (days between @2012-01 and @2012-02)
define D1134: (years between @2012 and @2014) > (days between @2012-01 and @2012-02)
define D1135: (years between @2012 and @2014) >= (days between @2012-01 and @2012-02)
define D1136: (years between @2012 and @2014) = (days between @2012-01 and @2012-02)
define D1137: (years between @2012 and @2014) < (months between DateTime(2005) and DateTime(2006, 7))
define D1138: (years between @2012 and @2014) <= (months between DateTime(2005) and DateTime(2006, 7))
define D1139: (years between @2012 and @2014) > (months between DateTime(2005) and DateTime(2006, 7))
define D1140: (years between @2012 and @2014) >= (months between DateTime(2005) and DateTime(2006, 7))
define D1141: (years between @2012 and @2014) = (months between DateTime(2005) and DateTime(2006, 7))
define D1142: (years between @2014 and @2012) < (years between @2014 and @2012)
define D1143: (years between @2014 and @2012) <= (years between @2014 and @2012)
define D1144: (years between @2014 and @2012) > (years between @2014 and @2012)
define D1145: (years between @2014 and @2012) >= (years between @2014 and @2012)
define D1146: (years between @2014 and @2012) = (years between @2014 and @2012)
define D1147: (years between @2014 and @2012) < (years between DateTime(2005) and DateTime(2010))
define D1148: (years between @2014 and @2012) <= (years between DateTime(2005) and DateTime(2010))
define D1149: (years between @2014 and @2012) > (years between DateTime(2005) and DateTime(2010))
define D1150: (years between @2014 and @2012) >= (years between DateTime(2005) and DateTime(2010))
define D1151: (years between @2014 and @2012) = (years between DateTime(2005) and DateTime(2010))
define D1152: (years between @2014 and @2012) < (days between @2017-08-07T17:00 and @2017-08-14T)
define D1153: (years between @2014 and @2012) <= (days between @2017-08-07T17:00 and @2017-08-14T)
define D1154: (years between @2014 and @2012) > (days between @2017-08-07T17:00 and @2017-08-14T)
define D1155: (years between @2014 and @2012) >= (days between @2017-08-07T17:00 and @2017-08-14T)
define D1156: (years between @2014 and @2012) = (days between @2017-08-07T17:00 and @2017-08-14T)
define D1157: (years between DateTime(2005) and DateTime(2010)) < (years between @2014 and @2012)
define D1158: (years between DateTime(2005) and DateTime(2010)) <= (years between @2014 and @2012)
define D1159: (years between DateTime(2005) and DateTime(2010)) > (years between @2014 and @2012)
define D1160: (years between DateTime(2005) and DateTime(2010)) >= (years between @2014 and @2012)
define D1161: (years between DateTime(2005) and DateTime(2010)) = (years between @2014 and @2012)
define D1162: (years between DateTime(2005) and DateTime(2010)) < (years between DateTime(2005) and DateTime(2010))
define D1163: (years between DateTime(2005) and DateTime(2010)) <= (years between DateTime(2005) and DateTime(2010))
define D1164: (years between DateTime(2005) and DateTime(2010)) > (years between DateTime(2005) and DateTime(2010))
define D1165: (years between DateTime(2005) and DateTime(2010)) >= (years between DateTime(2005) and DateTime(2010))
define D1166: (years between DateTime(2005) and DateTime(2010)) = (years between DateTime(2005) and DateTime(2010))
define D1167: (years between DateTime(2005) and DateTime(2010)) < (days between @2014-01-15 and @2014-02)
define D1168: (years between DateTime(2005) and DateTime(2010)) <= (days between @2014-01-15 and @2014-02)
define D1169: (years between DateTime(2005) and DateTime(2010)) > (days between @2014-01-15 and @2014-02)
define D1170: (years between DateTime(2005) and DateTime(2010)) >= (days between @2014-01-15 and @2014-02)
define D1171: (years between DateTime(2005) and DateTime(2010)) = (days between @2014-01-15 and @2014-02)
define D1172: (days between @2012-01 and @2012-02) < (years between @2012 and @2014)
define D1173: (days between @2012-01 and @2012-02) <= (years between @2012 and @2014)
define D1174: (days between @2012-01 and @2012-02) > (years between @2012 and @2014)
define D1175: (days between @2012-01 and @2012-02) >= (years between @2012 and @2014)
define D1176: (days between @2012-01 and @2012-02) = (years between @2012 and @2014)
define D1177: (days between @2012-01 and @2012-02) < (days between @2012-01 and @2012-02)
define D1178: (days between @2012-01 and @2012-02) <= (days between @2012-01 and @2012-02)
define D1179: (days between @2012-01 and @2012-02) > (days between @2012-01 and @2012-02)
define D1180: (days between @2012-01 and @2012-02) >= (days between @2012-01 and @2012-02)
define D1181: (days between @2012-01 and @2012-02) = (days between @2012-01 and @2012-02)
define D1182: (days between @2012-01 and @2012-02) < (months between DateTime(2005) and DateTime(2006, 7))
define D1183: (days between @2012-01 and @2012-02) <= (months between DateTime(2005) and DateTime(2006, 7))
define D1184: (days between @2012-01 and @2012-02) > (months between DateTime(2005) and DateTime(2006, 7))
define D1185: (days between @2012-01 and @2012-02) >= (months between DateTime(2005) and DateTime(2006, 7))
define D1186: (days between @2012-01 and @2012-02) = (months between DateTime(2005) and DateTime(2006, 7))
define D1187: (days between @2014-01-15 and @2014-02) < (years between DateTime(2005) and DateTime(2010))
define D1188: (days between @2014-01-15 and @2014-02) <= (years between DateTime(2005) and DateTime(2010))
define D1189: (days between @2014-01-15 and @2014-02) > (years between DateTime(2005) and DateTime(2010))
define D1190: (days between @2014-01-15 and @2014-02) >= (years between DateTime(2005) and DateTime(2010))
define D1191: (days between @2014-01-15 and @2014-02) = (years between DateTime(2005) and DateTime(2010))
define D1192: (days between @2014-01-15 and @2014-02) < (days between @2014-01-15 and @2014-02)
define D1193: (days between @2014-01-15 and @2014-02) <= (days between @2014-01-15 and @2014-02)
define D1194: (days between @2014-01-15 and @2014-02) > (days between @2014-01-15 and @2014-02)
define D1195: (days between @2014-01-15 and @2014-02) >= (days between @2014-01-15 and @2014-02)
define D1196: (days between @2014-01-15 and @2014-02) = (days between @2014-01-15 and @2014-02)
define D1197: (days between @2014-01-15 and @2014-02) < (days between @2017-08-07T17:00 and @2017-08-14T)
define D1198: (days between @2014-01-15 and @2014-02) <= (days between @2017-08-07T17:00 and @2017-08-14T)
define D1199: (days between @2014-01-15 and @2014-02) > (days between @2017-08-07T17:00 and @2017-08-14T)
define D1200: (days between @2014-01-15 and @2014-02) >= (days between @2017-08-07T17:00 and @2017-08-14T)
define D1201: (days between @2014-01-15 and @2014-02) = (days between @2017-08-07T17:00 and @2017-08-14T)
define D1202: (days between @2017-08-07T17:00 and @2017-08-14T) < (years between @2014 and @2012)
define D1203: (days between @2017-08-07T17:00 and @2017-08-14T) <= (years between @2014 and @2012)
define D1204: (days between @2017-08-07T17:00 and @2017-08-14T) > (years between @2014 and @2012)
define D1205: (days between @2017-08-07T17:00 and @2017-08-14T) >= (years between @2014 and @2012)
define D1206: (days between @2017-08-07T17:00 and @2017-08-14T) = (years between @2014 and @2012)
define D1207: (days between @2017-08-07T17:00 and @2017-08-14T) < (days between @2014-01-15 and @2014-02)
define D1208: (days between @2017-08-07T17:00 and @2017-08-14T) <= (days between @2014-01-15 and @2014-02)
define D1209: (days between @2017-08-07T17:00 and @2017-08-14T) > (days between @2014-01-15 and @2014-02)
define D1210: (days between @2017-08-07T17:00 and @2017-08-14T) >= (days between @2014-01-15 and @2014-02)
define D1211: (days between @2017-08-07T17:00 and @2017-08-14T) = (days between @2014-01-15 and @2014-02)
define D1212: (days between @2017-08-07T17:00 and @2017-08-14T) < (days between @2017-08-07T17:00 and @2017-08-14T)
define D1213: (days between @2017-08-07T17:00 and @2017-08-14T) <= (days between @2017-08-07T17:00 and @2017-08-14T)
define D1214: (days between @2017-08-07T17:00 and @2017-08-14T) > (days between @2017-08-07T17:00 and @2017-08-14T)
define D1215: (days between @2017-08-07T17:00 and @2017-08-14T) >= (days between @2017-08-07T17:00 and @2017-08-14T)
define D1216: (days between @2017-08-07T17:00 and @2017-08-14T) = (days between @2017-08-07T17:00 and @2017-08-14T)
define D1217: (months between DateTime(2005) and DateTime(2006, 7)) < (years between @2012 and @2014)
define D1218: (months between DateTime(2005) and DateTime(2006, 7)) <= (years between @2012 and @2014)
define D1219: (months between DateTime(2005) and DateTime(2006, 7)) > (years between @2012 and @2014)
define D1220: (months between DateTime(2005) and DateTime(2006, 7)) >= (years between @2012 and @2014)
define D1221: (months between DateTime(2005) and DateTime(2006, 7)) = (years between @2012 and @2014)
define D1222: (months between DateTime(2005) and DateTime(2006, 7)) < (days between @2012-01 and @2012-02)
define D1223: (months between DateTime(2005) and DateTime(2006, 7)) <= (days between @2012-01 and @2012-02)
define D1224: (months between DateTime(2005) and DateTime(2006, 7)) > (days between @2012-01 and @2012-02)
define D1225: (months between DateTime(2005) and DateTime(2006, 7)) >= (days between @2012-01 and @2012-02)
define D1226: (months between DateTime(2005) and DateTime(2006, 7)) = (days between @2012-01 and @2012-02)
define D1227: (months between DateTime(2005) and DateTime(2006, 7)) < (months between DateTime(2005) and DateTime(2006, 7))
define D1228: (months between DateTime(2005) and DateTime(2006, 7)) <= (months between DateTime(2005) and DateTime(2006, 7))
define D1229: (months between DateTime(2005) and DateTime(2006, 7)) > (months between DateTime(2005) and DateTime(2006, 7))
define D1230: (months between DateTime(2005) and DateTime(2006, 7)) >= (months between DateTime(2005) and DateTime(2006, 7))
define D1231: (months between DateTime(2005) and DateTime(2006, 7)) = (months between DateTime(2005) and DateTime(2006, 7))
define D1232: (years between @2012 and @2014) >= (years between @2011 and @2012)
define D1233: (years between @2011 and @2012) <= (years between @2012 and @2014)
define D1234: (years between @2012 and @2014) between (years between @2011 and @2012) and 5
// A DateTime to the day is read at the evaluation offset (-03:00 for this area),
// not at the offset of the other operand (plan 016, item 3).
define D1235: days between @2017-08-07T07:59+05:00 and @2017-08-14T
define D1236: days between @2017-08-14T and @2017-08-07T07:59+05:00
