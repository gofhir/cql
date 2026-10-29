library Differential version '1.0'
// Written by hand: the edges of docs/plans/013-null-donde-va-null.md. CQL
// 1.5.3, Appendix B: "operations that cause arithmetic overflow or underflow,
// or otherwise cannot be performed (such as division by 0) will result in
// null, rather than a run-time error"; Successor, "If the argument is already
// the maximum value for the type, a null is returned". Exp and Ln, which a
// float computes, are in limitsexp.cql, so these compare exactly.
define D0: successor of 2147483647
define D1: predecessor of -2147483648
define D2: successor of maximum Integer
define D3: predecessor of minimum Integer
define D4: successor of 2147483646
define D5: predecessor of -2147483647
define D6: successor of 9223372036854775807L
define D7: predecessor of (-9223372036854775807L - 1L)
define D8: successor of maximum Long
define D9: predecessor of minimum Long
define D10: successor of 2147483647L
define D11: successor of maximum Decimal
define D12: predecessor of minimum Decimal
define D13: successor of 99999999999999999999.99999999
define D14: predecessor of -99999999999999999999.99999999
define D15: successor of 99999999999999999999.99999998
define D16: predecessor of -99999999999999999999.99999998
define D17: successor of 99999999999999999999.99999999 'mg'
define D18: predecessor of -99999999999999999999.99999999 'mg'
define D19: successor of 99999999999999999999.99999998 'mg'
define D20: successor of @9999-12-31
define D21: successor of @9999-12
define D22: successor of @9999
define D23: predecessor of @0001-01-01
define D24: predecessor of @0001-01
define D25: predecessor of @0001
define D26: successor of maximum Date
define D27: predecessor of minimum Date
define D28: successor of @9999-12-30
define D29: predecessor of @0001-01-02
define D30: successor of DateTime(9999, 12, 31, 23, 59, 59, 999)
define D31: successor of DateTime(9999, 12, 31, 23, 59, 59)
define D32: successor of DateTime(9999, 12, 31, 23, 59)
define D33: successor of DateTime(9999, 12, 31, 23)
define D34: successor of DateTime(9999, 12, 31)
define D35: successor of DateTime(9999, 12)
define D36: successor of DateTime(9999)
define D37: predecessor of DateTime(0001, 1, 1, 0, 0, 0, 0)
define D38: predecessor of DateTime(0001, 1, 1)
define D39: predecessor of DateTime(1)
define D40: successor of DateTime(9999, 12, 31, 23, 59, 59, 999, 0.0)
define D41: successor of DateTime(9999, 12, 31, 23, 59, 59, 999, 14.0)
define D42: successor of DateTime(9999, 12, 31, 23, 59, 59, 999, -5.0)
define D43: predecessor of DateTime(0001, 1, 1, 0, 0, 0, 0, -12.0)
define D44: predecessor of DateTime(0001, 1, 1, 0, 0, 0, 0, 14.0)
define D45: successor of maximum DateTime
define D46: predecessor of minimum DateTime
define D47: successor of DateTime(9999, 12, 31, 23, 59, 59, 998, -3.0)
define D48: successor of @T23:59:59.999
define D49: successor of @T23:59:59
define D50: successor of @T23:59
define D51: successor of @T23
define D52: predecessor of @T00:00:00.000
define D53: predecessor of @T00:00:00
define D54: predecessor of @T00:00
define D55: predecessor of @T00
define D56: successor of maximum Time
define D57: predecessor of minimum Time
define D58: successor of @T23:59:59.998
define D59: predecessor of @T00:00:00.001
define D60: Coalesce(successor of maximum Integer, 1)
define D61: (successor of @T23:59:59.999) is null
define D62: (predecessor of @0001-01-01) is null
define D63: start of Interval(2147483647, null]
define D64: end of Interval[null, -2147483648)
define D65: start of Interval(maximum Long, null]
define D66: start of Interval(maximum Decimal, null]
define D67: end of Interval[null, minimum Decimal)
define D68: start of Interval(@9999-12-31, null]
define D70: end of Interval[null, @0001-01-01T00:00:00.000)
define D71: start of Interval(@T23:59:59.999, null]
define D72: end of Interval[null, @T00:00:00.000)
define D73: start of Interval(2147483646, null]
define D74: end of Interval[null, @T00:00:00.001)
define D75: Interval[maximum Decimal, maximum Decimal] meets before Interval[maximum Decimal, maximum Decimal]
define D76: Interval[maximum Decimal, maximum Decimal] meets Interval[maximum Decimal, maximum Decimal]
define D77: Interval[99999999999999999999.99999998, 99999999999999999999.99999998] meets before Interval[maximum Decimal, maximum Decimal]
define D78: Interval[maximum Integer, maximum Integer] meets before Interval[maximum Integer, maximum Integer]
define D79: Interval[@T23:59:59.999, @T23:59:59.999] meets before Interval[@T23:59:59.999, @T23:59:59.999]
define D80: Interval[@T23:59:59.998, @T23:59:59.998] meets before Interval[@T23:59:59.999, @T23:59:59.999]
define D81: Interval[@9999-12-31, @9999-12-31] meets before Interval[@9999-12-31, @9999-12-31]
define D82: Interval[99999999999999999999.99999999 'g', 99999999999999999999.99999999 'g'] meets before Interval[99999999999999999999.99999999 'g', 99999999999999999999.99999999 'g']
define D83: Interval[0.0, maximum Decimal] except Interval[0.0, maximum Decimal)
define D84: collapse { Interval[0.0, 99999999999999999999.99999998], Interval[maximum Decimal, maximum Decimal] }
define D85: Interval[0.0, maximum Decimal] = Interval[0.0, maximum Decimal]
define D109: 5.0 'g' / 0
define D110: 5.0 'g' / 0.0
define D111: 5 'g' / 0
define D112: (5.0 'g' / 0) + 1 'g'
define D113: 5.0 'g' / 2
define D114: 5.0 'g' / 0 'g'
define D115: 5 'g' / 0.0 'mg'
define D116: 5.0 / 0
define D117: 5 / 0
define D118: 5.0 'g' mod 0 'g'
define D119: 5.0 'g' div 0 'g'
