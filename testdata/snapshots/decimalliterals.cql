library Differential version '1.0'
// Written by hand: the Decimal literals of docs/plans/011-literal-decimal.md.
// CQL (Types, Decimal): "a precision of 28 and a scale of 8", and
// "implementations must support at least the 28 digits of precision and 8
// digits of scale, but may support more precise values". cql-tests,
// ValueLiteralsAndSelectors.xml: DecimalTenthStep (0.000000001) and
// Decimal10Pow28 (10000000000000000000000000000.00000000) are invalid,
// "Either some error saying the literal uses more precision than the Decimal
// step size, or round to a whole step".
define D0: 1.00000000
define D1: 1.000000005
define D2: 0.000000001
define D3: +0.000000001
define D4: -0.000000001
define D5: 0.100000001
define D6: 0.0000000010
define D7: 12345678901234567890.123456789
define D8: 1.000000000
define D9: 1.0000000050
define D10: 0.100000000
define D11: 1.00000000000000000000000000000
define D12: 0.000000000
define D13: -0.00000000
define D14: -0.000000000
define D15: -0.0
define D16: 000000000000000000000000000001.0
define D17: 0000000000000000000000000000000000.5
define D18: 99999999999999999999.99999999
define D19: 123456789012345678901.0
define D20: 1234567890123456789012345678.0
define D21: 9999999999999999999999999999.99999999
define D22: 10000000000000000000000000000.0
define D23: -10000000000000000000000000000.0
define D24: 100000000000000000000000000000.0000
define D25: 1.000000005 + 0
define D26: 1.000000005 - 0
define D27: 1.000000005 * 1
define D28: 1.000000005 = 1.00000000
define D29: Round(1.000000005, 9)
define D30: Truncate(1.000000005)
define D31: Abs(-0.000000001)
define D32: -(1.000000005)
define D33: 1.000000000 + 0
define D34: 1.000000000 = 1.0
define D35: 1.000000005 is Decimal
define D36: Coalesce(null, 1.000000005)
define D37: 1.000000005 in {1.0}
define D38: ToString(1.000000000)
define D39: ToString(000000000000000000000000000001.0)
define D40: {1.0, 1.000000005}
define D41: Interval[1.0, 1.000000005]
define D42: Count({1.0, 0.000000001})
define D43: if false then 0.000000001 else 1.0
define D44: if true then 0.000000001 else 1.0
define D45: false and (0.000000001 = 1.0)
define D46: null and (0.000000001 = 1.0)
define D47: case when 1 = 2 then 0.000000001 else 2.0 end
define D48: ToDecimal('1.000000005')
define D49: ToDecimal('1.000000000')
define D50: ToDecimal('10000000000000000000000000000.0')
define D51: ToDecimal('123456789012345678901.0')
define D52: maximum Decimal
define D53: minimum Decimal
define D54: 123456789012345678901.0 > maximum Decimal
define D55: 5.999999999 'g'
define D56: 1.000000005 'mg'
define D57: 1.000000000 'mg'
define D58: 0.000000001 'mg'
define D59: 10000000000000000000000000000.0 'mg'
