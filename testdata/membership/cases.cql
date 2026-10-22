library Membership version '1.0'

define NullList: null as List<Integer>
define D0: (null as Integer) in { 1, 2 }
define D1: { 1, 2 } contains (null as Integer)
define D2: null in { 1, 2 }
define D3: { 1, 2 } contains null
define D4: (null as Integer) in { 1, null }
define D5: { 1, null } contains (null as Integer)
define D6: (null as Integer) in ({} as List<Integer>)
define D7: ({} as List<Integer>) contains (null as Integer)
define D8: {} contains null
define D9: null in {}
define D10: { 1, 2 } includes (null as Integer)
define D11: (null as Integer) included in { 1, 2 }
define D12: { 1, null } includes (null as Integer)
define D13: (null as Integer) included in { 1, null }
define D14: 1 in (null as List<Integer>)
define D15: (null as List<Integer>) contains 1
define D16: (null as List<Integer>) contains (null as Integer)
define D17: (null as Integer) in (null as List<Integer>)
define D18: (null as String) in { 'a' }
define D19: (null as Code) in { Code { code: 'x' } }
define D20: (null as Interval<Integer>) in { Interval[1, 2] }
define D21: { Interval[1, 2] } contains (null as Interval<Integer>)
define D22: (null as Date) in { @2020-01-01 }
define D23: { @2020-01-01 } contains (null as Date)
define D24: (null as Tuple { a Integer }) in { Tuple { a: 1 } }
define D25: { { 1 } } contains (null as List<Integer>)
define D26: not ((null as Integer) in { 1, 2 })
define D27: (null as Interval<Integer>) contains (null as Integer)
define D28: (null as Interval<Integer>) contains 1
define D29: (null as Integer) in (null as Interval<Integer>)
define D30: 1 in (null as Interval<Integer>)
define D31: Interval[1, 2] contains (null as Integer)
define D32: (null as Integer) in Interval[1, 2]
define D33: (null as List<Integer>) includes (null as Integer)
define D34: (null as Integer) included in (null as List<Integer>)
define D35: { (null as Integer) } contains (null as Integer)
define D36: List<Integer>{} contains (null as Integer)
define D37: { 's', 'a', 'm' } includes (null as List<String>)
define D38: { 's', 'a', 'm' } includes (null as String)
define D39: (null as List<String>) included in { 's' }
define D40: (null as String) included in { 's' }
define D41: (null as List<Integer>) includes 1
define D42: 1 included in (null as List<Integer>)
define D43: { { 1 } } includes (null as List<Integer>)
define D44: (null as List<Integer>) included in { { 1 } }
define D45: (null as Interval<Integer>) includes 1
define D46: 1 included in (null as Interval<Integer>)
define D47: 1 during (null as Interval<Integer>)
define D48: (null as Integer) during { 1, 2 }
define D49: { Interval[1, null] } contains (null as Interval<Integer>)
define D50: { Tuple { a: null } } contains (null as Tuple { a Integer })
define D51: { { null as Integer } } contains (null as List<Integer>)
define D52: First(List<Integer>{}) in { 1, 2 }
define D53: NullList includes (null as Integer)
define D54: (null as List<String>) included in { 'b', null }
define D55: { 'a', null } includes (null as List<String>)
define D56: (null as Integer) included in List<Decimal>{ 1.0 }
define D57: (null as Integer) included in List<Decimal>{ 1.0, null }
define D58: List<Decimal>{ 1.0 } includes (null as Integer)
define D59: 1 included in (null as Interval<Decimal>)
define D60: (null as Integer) during (null as Interval<Decimal>)
define D61: (null as Date) included in (null as Interval<DateTime>)
define D62: (null as Interval<DateTime>) includes day of @2020-01-01T00:00:00
define D63: (null as Decimal) included in List<Integer>{ 1, 2 }
define D64: (null as Decimal) included in List<Integer>{ 1, null }
define D65: (null as List<Integer>) includes (null as Decimal)
define D66: (null as List<Integer>) included in List<List<Decimal>>{ { 1.0 } }
define D67: (null as List<Integer>) included in List<List<Decimal>>{ { 1.0 }, null }
define D68: List<List<Decimal>>{ { 1.0 } } includes (null as List<Integer>)
define D69: (null as Decimal) during (null as Interval<Integer>)
define D70: { 1, 2 } includes (null as Decimal)
define D71: (null as DateTime) included in { @2020-01-01 }
define D72: 1.0 included in (null as Interval<Integer>)
define D73: (null as Interval<Date>) includes @2020-01-01T00:00:00
define D74: (null as List<Integer>) included in List<Decimal>{ 1.0, null }
define D75: (null as List<Decimal>) included in List<Integer>{ 1, null }
