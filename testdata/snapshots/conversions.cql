library Differential version '1.0'
using FHIR version '4.0.1'
include FHIRHelpers version '4.0.1'
// Written by hand: the strings and values of docs/plans/014-todate-toratio.md,
// read by ToDate, ConvertsToDate, ToRatio, ConvertsToRatio, convert and the
// Ratio literal. CQL 1.5.3, Appendix B, ToDate: "time portion will be
// ignored"; ToRatio: "<quantity>:<quantity> where <quantity> is the format used
// to by the ToQuantity operator". ConvertsToRatio of a Ratio is left out: the
// Java engine answers the Ratio itself, which its harness cannot encode. A
// type test names System's type: under `using FHIR` the reference translator
// reads a bare Ratio or Quantity as FHIR's (plan 014). The gaps in the
// numbering are definitions that were there twice.
define D0: ToDate('2014-01-01')
define D1: ConvertsToDate('2014-01-01')
define D2: ToDate('2014-01')
define D3: ConvertsToDate('2014-01')
define D4: ToDate('2014')
define D5: ConvertsToDate('2014')
define D6: ToDate('2014-01-01T12:05:05.955')
define D7: ConvertsToDate('2014-01-01T12:05:05.955')
define D8: ToDate('2014-01-01T12:30:00')
define D9: ConvertsToDate('2014-01-01T12:30:00')
define D10: ToDate('2014-01-01T')
define D11: ConvertsToDate('2014-01-01T')
define D12: ToDate('2014-01-01T25:00:00')
define D13: ConvertsToDate('2014-01-01T25:00:00')
define D14: ToDate('2014-01-01T12:30:00Z')
define D15: ConvertsToDate('2014-01-01T12:30:00Z')
define D16: ToDate('2014-01-01T12:30:00-05:00')
define D17: ConvertsToDate('2014-01-01T12:30:00-05:00')
define D18: ToDate('2014-01-01T23:30:00+14:00')
define D19: ConvertsToDate('2014-01-01T23:30:00+14:00')
define D20: ToDate('2014-01-01T12')
define D21: ConvertsToDate('2014-01-01T12')
define D22: ToDate('2014-01-01T12:30')
define D23: ConvertsToDate('2014-01-01T12:30')
define D24: ToDate('2014-01-01T12:30:00.1234')
define D25: ConvertsToDate('2014-01-01T12:30:00.1234')
define D26: ToDate('2014-01-01Tabc')
define D27: ConvertsToDate('2014-01-01Tabc')
define D28: ToDate('2014-02-30')
define D29: ConvertsToDate('2014-02-30')
define D30: ToDate('2014-02-30T10:00')
define D31: ConvertsToDate('2014-02-30T10:00')
define D32: ToDate('2014-13')
define D33: ConvertsToDate('2014-13')
define D34: ToDate('2014-1-1')
define D35: ConvertsToDate('2014-1-1')
define D36: ToDate('2014-1')
define D37: ConvertsToDate('2014-1')
define D38: ToDate(' 2014-01-01')
define D39: ConvertsToDate(' 2014-01-01')
define D40: ToDate('2014-01-01 ')
define D41: ConvertsToDate('2014-01-01 ')
define D42: ToDate('20140101')
define D43: ConvertsToDate('20140101')
define D44: ToDate('2014/01/01')
define D45: ConvertsToDate('2014/01/01')
define D46: ToDate('0001-01-01')
define D47: ConvertsToDate('0001-01-01')
define D48: ToDate('9999-12-31')
define D49: ConvertsToDate('9999-12-31')
define D50: ToDate('0000-01-01')
define D51: ConvertsToDate('0000-01-01')
define D52: ToDate('2014-01-01Z')
define D53: ConvertsToDate('2014-01-01Z')
define D54: ToDate('2014-01-01+01:00')
define D55: ConvertsToDate('2014-01-01+01:00')
define D56: ToDate('')
define D57: ConvertsToDate('')
define D58: ToDate('abc')
define D59: ConvertsToDate('abc')
define D60: ToDate('2014-01-01T12:30:00+25:00')
define D61: ConvertsToDate('2014-01-01T12:30:00+25:00')
define D62: ToDate('2016-02-29')
define D63: ConvertsToDate('2016-02-29')
define D64: ToDate('2015-02-29')
define D65: ConvertsToDate('2015-02-29')
define D66: ToDate('2014-01-01T24:00:00')
define D67: ConvertsToDate('2014-01-01T24:00:00')
define D68: ToDate('2014-01-01T12:60')
define D69: ConvertsToDate('2014-01-01T12:60')
define D70: ToDate(null as String)
define D71: ConvertsToDate(null as String)
define D72: ToDate(DateTime(2014, 1, 1, 23, 0, 0, 0, -5.0))
define D73: ToDate(DateTime(2014))
define D74: ToDate(DateTime(2014, 1))
define D75: ToDate(@2014-01-01T23:59:59.999+14:00)
define D76: ToDate(@2014-01-01T00:00:00.000-12:00)
define D77: ToDate(@2014-01-01)
define D78: ConvertsToDate(@2014-01-01T10:00)
define D79: ConvertsToDate(@2014-01-01)
define D80: ConvertsToDate(5)
define D81: convert '2014-01-01T12:00:00' to Date
define D82: convert '2014-01' to Date
define D83: ToDate('2014-01-01T12:30:00') = @2014-01-01
define D84: ToDate('2014-01') = @2014-01
define D85: ToDate(DateTime(2014, 1, 1, 12)) = @2014-01-01
define D86: ToRatio('1.0 \'mg\':2.0 \'mg\'') is System.Ratio
define D87: ToRatio('1.0 \'mg\':2.0 \'mg\'').numerator
define D88: ToRatio('1.0 \'mg\':2.0 \'mg\'').denominator
define D89: ConvertsToRatio('1.0 \'mg\':2.0 \'mg\'')
define D90: ToRatio('1.0 \'mg\';2.0 \'mg\'') is System.Ratio
define D91: ToRatio('1.0 \'mg\';2.0 \'mg\'').numerator
define D92: ToRatio('1.0 \'mg\';2.0 \'mg\'').denominator
define D93: ConvertsToRatio('1.0 \'mg\';2.0 \'mg\'')
define D94: ToRatio('1:2') is System.Ratio
define D95: ToRatio('1:2').numerator
define D96: ToRatio('1:2').denominator
define D97: ConvertsToRatio('1:2')
define D98: ToRatio('1 : 2') is System.Ratio
define D99: ToRatio('1 : 2').numerator
define D100: ToRatio('1 : 2').denominator
define D101: ConvertsToRatio('1 : 2')
define D102: ToRatio('1.0 \'mg\' : 2.0 \'mg\'') is System.Ratio
define D103: ToRatio('1.0 \'mg\' : 2.0 \'mg\'').numerator
define D104: ToRatio('1.0 \'mg\' : 2.0 \'mg\'').denominator
define D105: ConvertsToRatio('1.0 \'mg\' : 2.0 \'mg\'')
define D106: ToRatio('1 \'mg\':2 \'mL\'') is System.Ratio
define D107: ToRatio('1 \'mg\':2 \'mL\'').numerator
define D108: ToRatio('1 \'mg\':2 \'mL\'').denominator
define D109: ConvertsToRatio('1 \'mg\':2 \'mL\'')
define D110: ToRatio('1:0') is System.Ratio
define D111: ToRatio('1:0').numerator
define D112: ToRatio('1:0').denominator
define D113: ConvertsToRatio('1:0')
define D114: ToRatio('0:0') is System.Ratio
define D115: ToRatio('0:0').numerator
define D116: ToRatio('0:0').denominator
define D117: ConvertsToRatio('0:0')
define D118: ToRatio('1') is System.Ratio
define D119: ToRatio('1').numerator
define D120: ToRatio('1').denominator
define D121: ConvertsToRatio('1')
define D122: ToRatio('1:2:3') is System.Ratio
define D123: ToRatio('1:2:3').numerator
define D124: ToRatio('1:2:3').denominator
define D125: ConvertsToRatio('1:2:3')
define D126: ToRatio(':2') is System.Ratio
define D127: ToRatio(':2').numerator
define D128: ToRatio(':2').denominator
define D129: ConvertsToRatio(':2')
define D130: ToRatio('1:') is System.Ratio
define D131: ToRatio('1:').numerator
define D132: ToRatio('1:').denominator
define D133: ConvertsToRatio('1:')
define D134: ToRatio('') is System.Ratio
define D135: ToRatio('').numerator
define D136: ToRatio('').denominator
define D137: ConvertsToRatio('')
define D138: ToRatio(':') is System.Ratio
define D139: ToRatio(':').numerator
define D140: ToRatio(':').denominator
define D141: ConvertsToRatio(':')
define D142: ToRatio(' 1:2') is System.Ratio
define D143: ToRatio(' 1:2').numerator
define D144: ToRatio(' 1:2').denominator
define D145: ConvertsToRatio(' 1:2')
define D146: ToRatio('1:2 ') is System.Ratio
define D147: ToRatio('1:2 ').numerator
define D148: ToRatio('1:2 ').denominator
define D149: ConvertsToRatio('1:2 ')
define D150: ToRatio('-1:2') is System.Ratio
define D151: ToRatio('-1:2').numerator
define D152: ToRatio('-1:2').denominator
define D153: ConvertsToRatio('-1:2')
define D154: ToRatio('1:-2') is System.Ratio
define D155: ToRatio('1:-2').numerator
define D156: ToRatio('1:-2').denominator
define D157: ConvertsToRatio('1:-2')
define D158: ToRatio('1e2:3') is System.Ratio
define D159: ToRatio('1e2:3').numerator
define D160: ToRatio('1e2:3').denominator
define D161: ConvertsToRatio('1e2:3')
define D162: ToRatio('1 \'foo\':2') is System.Ratio
define D163: ToRatio('1 \'foo\':2').numerator
define D164: ToRatio('1 \'foo\':2').denominator
define D165: ConvertsToRatio('1 \'foo\':2')
define D166: ToRatio('1 \'{a:b}\':2') is System.Ratio
define D167: ToRatio('1 \'{a:b}\':2').numerator
define D168: ToRatio('1 \'{a:b}\':2').denominator
define D169: ConvertsToRatio('1 \'{a:b}\':2')
define D170: ToRatio('1.5 \'mg/dL\':1 \'1\'') is System.Ratio
define D171: ToRatio('1.5 \'mg/dL\':1 \'1\'').numerator
define D172: ToRatio('1.5 \'mg/dL\':1 \'1\'').denominator
define D173: ConvertsToRatio('1.5 \'mg/dL\':1 \'1\'')
define D174: ToRatio('1 \'mg\' :2 \'mg\'') is System.Ratio
define D175: ToRatio('1 \'mg\' :2 \'mg\'').numerator
define D176: ToRatio('1 \'mg\' :2 \'mg\'').denominator
define D177: ConvertsToRatio('1 \'mg\' :2 \'mg\'')
define D178: ToRatio('1 \'mg\': 2 \'mg\'') is System.Ratio
define D179: ToRatio('1 \'mg\': 2 \'mg\'').numerator
define D180: ToRatio('1 \'mg\': 2 \'mg\'').denominator
define D181: ConvertsToRatio('1 \'mg\': 2 \'mg\'')
define D182: ToRatio('1\'mg\':2\'mg\'') is System.Ratio
define D183: ToRatio('1\'mg\':2\'mg\'').numerator
define D184: ToRatio('1\'mg\':2\'mg\'').denominator
define D185: ConvertsToRatio('1\'mg\':2\'mg\'')
define D186: ToRatio('1 mg:2 mg') is System.Ratio
define D187: ToRatio('1 mg:2 mg').numerator
define D188: ToRatio('1 mg:2 mg').denominator
define D189: ConvertsToRatio('1 mg:2 mg')
define D190: ToRatio('1 day:2 days') is System.Ratio
define D191: ToRatio('1 day:2 days').numerator
define D192: ToRatio('1 day:2 days').denominator
define D193: ConvertsToRatio('1 day:2 days')
define D194: ToRatio('1 \'g\':1000 \'mg\'') is System.Ratio
define D195: ToRatio('1 \'g\':1000 \'mg\'').numerator
define D196: ToRatio('1 \'g\':1000 \'mg\'').denominator
define D197: ConvertsToRatio('1 \'g\':1000 \'mg\'')
define D198: ToRatio('99999999999999999999.99999999:1') is System.Ratio
define D199: ToRatio('99999999999999999999.99999999:1').numerator
define D200: ToRatio('99999999999999999999.99999999:1').denominator
define D201: ConvertsToRatio('99999999999999999999.99999999:1')
define D202: ToRatio('1.123456789:1') is System.Ratio
define D203: ToRatio('1.123456789:1').numerator
define D204: ToRatio('1.123456789:1').denominator
define D205: ConvertsToRatio('1.123456789:1')
define D206: ToRatio(null as String)
define D207: ConvertsToRatio(null as String)
define D208: ToRatio('1 \'mg\':2 \'mg\'') = 1 'mg':2 'mg'
define D209: ToRatio('1:2') = 1:2
define D210: ToRatio('1 \'mg\':2 \'mg\'') ~ 1 'mg':2 'mg'
define D211: ToString(ToRatio('1.0 \'mg\':2.0 \'mg\''))
define D212: ToString(1 'mg':2 'mg')
define D213: ConvertsToRatio(5)
define D214: ToRatio('1:2').numerator is System.Quantity
define D215: 1:2 is System.Ratio
define D216: (1 'mg':2 'mg').numerator
define D217: (1 'mg':2 'mg').denominator
define D218: (1.0 'mg':2.0 'mg').numerator
define D219: ToString(1.0 'mg':2.0 'mg')
define D220: ToString(ToRatio('1 \'mg\':2 \'mg\''))
define D221: ToString(1.0 'mg')
define D222: (1:2) = (2:4)
define D223: (1:2) ~ (2:4)
define D224: (1 'g':2 'mg') = (1000 'mg':2 'mg')
define D225: ToRatio('1:2') = (1:2)
define D226: (convert '1:2' to System.Ratio) is System.Ratio
define D227: (convert '1:2' to System.Ratio).numerator
define D228: convert 'abc' to System.Ratio
define D230: convert 'abc' to Date
define D231: convert '2014/01/01' to Date
define D232: convert @2014-01-01T10:00 to Date
define D233: convert '2014-01-01' to System.Date
define D234: cast '2014-01-01' as Date
define D235: '1:2' as System.Ratio
define D238: ToDateTime('2014-01-01T')
define D239: ToDate('2014T12:30')
define D240: ToDate('2014-01T12:30')
define D241: ToDateTime('2014-01T12:30')
define D242: ToDate('2014-01-01T12:30:00+14:01')
define D243: ToDate('2014-01-01T23:59:60')
define D244: ToDate('2014-01-01T00:30:00+14:00')
define D245: ToDate('2014-01-01T00:30:00-12:00')
define D246: ToRatio('\t1:2').numerator
define D247: ToRatio('1\n:2').numerator
define D248: ToRatio('1 \'mg:2') is null
define D249: ToRatio('1:2 \'{a:b}\'').denominator
define D250: ToRatio('1L:2') is null
define D251: ToRatio('1 \'mg\':2 \'mg\':3') is null
define D252: ToRatio('1::2') is null
define D253: ToRatio('1 \'m\'\'g\':2') is null
define D254: ConvertsToRatio('Inf:NaN')
define D255: ConvertsToRatio('1:128')
define D256: ConvertsToDate(FHIR.date { value: @2014-01-01 })
define D257: ConvertsToDate(FHIR.dateTime { value: @2014-01-01T10:00 })
define D258: ToDate(FHIR.date { value: @2014-01-01 })
define D259: ToRatio(FHIR.Ratio { numerator: FHIR.Quantity { value: FHIR.decimal { value: 1.0 } }, denominator: FHIR.Quantity { value: FHIR.decimal { value: 2.0 } } }) is System.Ratio
define D260: ConvertsToDate(1 'mg')
define D261: ConvertsToRatio(1 'mg')
define D262: ConvertsToDate(@T12:00)
define D263: (1 day:2 days).numerator
define D264: (1 'foo':2 'foo').numerator
define D265: (1:2).numerator
define D266: ToString(1 'day':2 'days')
define D267: ToRatio(ToString(1 'day':2 'days')).numerator
