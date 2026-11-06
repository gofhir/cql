library Differential version '1.0'
using FHIR version '4.0.1'
include FHIRHelpers version '4.0.1'
parameter MP Interval<DateTime> default Interval[@2020-03-01, @2020-03-05]
parameter P DateTime default @2020-03-01
define function F(x DateTime): x
define function G(x Interval<DateTime>): x
define D0: @2020-03-01 = @2020-03-01T10:00:00.000
define D1: @2020-03-01T10:00:00.000 = @2020-03-01
define D2: @2020-03-01 != @2020-03-01T10:00:00.000
define D3: @2020-03-01T10:00:00.000 != @2020-03-01
define D4: @2020-03-01 ~ @2020-03-01T10:00:00.000
define D5: @2020-03-01T10:00:00.000 ~ @2020-03-01
define D6: @2020-03-01 !~ @2020-03-01T10:00:00.000
define D7: @2020-03-01T10:00:00.000 !~ @2020-03-01
define D8: @2020-03-01 < @2020-03-01T10:00:00.000
define D9: @2020-03-01T10:00:00.000 < @2020-03-01
define D10: @2020-03-01 >= @2020-03-01T10:00:00.000
define D11: @2020-03-01T10:00:00.000 >= @2020-03-01
define D12: @2020-03-01 between @2020-03-01T10:00:00.000 and @2020-03-01T10:00:00.000
define D13: @2020-03-01T10:00:00.000 between @2020-03-01 and @2020-03-01
define D14: @2020-03-01 same day as @2020-03-01T10:00:00.000
define D15: @2020-03-01T10:00:00.000 same day as @2020-03-01
define D16: @2020-03-01 before @2020-03-01T10:00:00.000
define D17: @2020-03-01T10:00:00.000 before @2020-03-01
define D18: @2020-03-01 after day of @2020-03-01T10:00:00.000
define D19: @2020-03-01T10:00:00.000 after day of @2020-03-01
define D20: @2020-03-01 same or before @2020-03-01T10:00:00.000
define D21: @2020-03-01T10:00:00.000 same or before @2020-03-01
define D22: @2020-03-01 within 3 days of @2020-03-01T10:00:00.000
define D23: @2020-03-01T10:00:00.000 within 3 days of @2020-03-01
define D24: @2020-03-01 1 day or less before @2020-03-01T10:00:00.000
define D25: @2020-03-01T10:00:00.000 1 day or less before @2020-03-01
define D26: @2020-03-01 in Interval[@2020-03-01T10:00:00.000, @2020-03-05T10:00:00.000]
define D28: @2020-03-01 during Interval[@2020-03-01T10:00:00.000, @2020-03-05T10:00:00.000]
define D30: Interval[@2020-03-01T10:00:00.000, @2020-03-05T10:00:00.000] includes @2020-03-01
define D32: Interval[@2020-03-01T10:00:00.000, @2020-03-05T10:00:00.000] contains @2020-03-01
define D34: @2020-03-01 in { @2020-03-01T10:00:00.000 }
define D35: @2020-03-01T10:00:00.000 in { @2020-03-01 }
define D36: { @2020-03-01T10:00:00.000 } contains @2020-03-01
define D37: { @2020-03-01 } contains @2020-03-01T10:00:00.000
define D38: { @2020-03-01, @2020-03-01T10:00:00.000 }
define D39: { @2020-03-01T10:00:00.000, @2020-03-01 }
define D40: Interval[@2020-03-01, @2020-03-01T10:00:00.000]
define D41: Interval[@2020-03-01T10:00:00.000, @2020-03-01]
define D42: if true then @2020-03-01 else @2020-03-01T10:00:00.000
define D43: if true then @2020-03-01T10:00:00.000 else @2020-03-01
define D44: case when true then @2020-03-01 else @2020-03-01T10:00:00.000 end
define D45: case when true then @2020-03-01T10:00:00.000 else @2020-03-01 end
define D46: Coalesce(@2020-03-01, @2020-03-01T10:00:00.000)
define D47: Coalesce(@2020-03-01T10:00:00.000, @2020-03-01)
define D48: Coalesce(null, @2020-03-01, @2020-03-01T10:00:00.000)
define D49: Coalesce(null, @2020-03-01T10:00:00.000, @2020-03-01)
define D50: Coalesce(@2020-03-01, null, @2020-03-01T10:00:00.000)
define D51: Coalesce({ @2020-03-01 }, { @2020-03-01T10:00:00.000 })
define D52: Interval[@2020-03-01, @2020-03-05] overlaps Interval[@2020-03-01T10:00:00.000, @2020-03-05T10:00:00.000]
define D53: Interval[@2020-03-01T10:00:00.000, @2020-03-05T10:00:00.000] overlaps Interval[@2020-03-01, @2020-03-05]
define D54: Interval[@2020-03-01, @2020-03-05] meets Interval[@2020-03-01T10:00:00.000, @2020-03-05T10:00:00.000]
define D55: Interval[@2020-03-01T10:00:00.000, @2020-03-05T10:00:00.000] meets Interval[@2020-03-01, @2020-03-05]
define D56: Interval[@2020-03-01, @2020-03-05] starts Interval[@2020-03-01T10:00:00.000, @2020-03-05T10:00:00.000]
define D57: Interval[@2020-03-01T10:00:00.000, @2020-03-05T10:00:00.000] starts Interval[@2020-03-01, @2020-03-05]
define D58: Interval[@2020-03-01, @2020-03-05] ends Interval[@2020-03-01T10:00:00.000, @2020-03-05T10:00:00.000]
define D59: Interval[@2020-03-01T10:00:00.000, @2020-03-05T10:00:00.000] ends Interval[@2020-03-01, @2020-03-05]
define D60: Interval[@2020-03-01, @2020-03-05] during Interval[@2020-03-01T10:00:00.000, @2020-03-05T10:00:00.000]
define D61: Interval[@2020-03-01T10:00:00.000, @2020-03-05T10:00:00.000] during Interval[@2020-03-01, @2020-03-05]
define D62: Interval[@2020-03-01, @2020-03-05] includes Interval[@2020-03-01T10:00:00.000, @2020-03-05T10:00:00.000]
define D63: Interval[@2020-03-01T10:00:00.000, @2020-03-05T10:00:00.000] includes Interval[@2020-03-01, @2020-03-05]
define D64: Interval[@2020-03-01, @2020-03-05] properly includes Interval[@2020-03-01T10:00:00.000, @2020-03-05T10:00:00.000]
define D65: Interval[@2020-03-01T10:00:00.000, @2020-03-05T10:00:00.000] properly includes Interval[@2020-03-01, @2020-03-05]
define D66: Interval[@2020-03-01, @2020-03-05] = Interval[@2020-03-01T10:00:00.000, @2020-03-05T10:00:00.000]
define D67: Interval[@2020-03-01T10:00:00.000, @2020-03-05T10:00:00.000] = Interval[@2020-03-01, @2020-03-05]
define D68: Interval[@2020-03-01, @2020-03-05] union Interval[@2020-03-01T10:00:00.000, @2020-03-05T10:00:00.000]
define D69: Interval[@2020-03-01T10:00:00.000, @2020-03-05T10:00:00.000] union Interval[@2020-03-01, @2020-03-05]
define D70: Interval[@2020-03-01, @2020-03-05] intersect Interval[@2020-03-01T10:00:00.000, @2020-03-05T10:00:00.000]
define D71: Interval[@2020-03-01T10:00:00.000, @2020-03-05T10:00:00.000] intersect Interval[@2020-03-01, @2020-03-05]
define D72: Interval[@2020-03-01, @2020-03-05] except Interval[@2020-03-01T10:00:00.000, @2020-03-05T10:00:00.000]
define D73: Interval[@2020-03-01T10:00:00.000, @2020-03-05T10:00:00.000] except Interval[@2020-03-01, @2020-03-05]
define D74: Interval[@2020-03-01, @2020-03-05] overlaps day of Interval[@2020-03-01T10:00:00.000, @2020-03-05T10:00:00.000]
define D75: Interval[@2020-03-01T10:00:00.000, @2020-03-05T10:00:00.000] overlaps day of Interval[@2020-03-01, @2020-03-05]
define D76: { @2020-03-01 } union { @2020-03-01T10:00:00.000 }
define D77: { @2020-03-01T10:00:00.000 } union { @2020-03-01 }
define D78: { @2020-03-01 } intersect { @2020-03-01T10:00:00.000 }
define D79: { @2020-03-01T10:00:00.000 } intersect { @2020-03-01 }
define D80: { @2020-03-01 } except { @2020-03-01T10:00:00.000 }
define D81: { @2020-03-01T10:00:00.000 } except { @2020-03-01 }
define D82: { @2020-03-01 } = { @2020-03-01T10:00:00.000 }
define D83: { @2020-03-01T10:00:00.000 } = { @2020-03-01 }
define D84: days between @2020-03-01 and @2020-03-01T10:00:00.000
define D85: days between @2020-03-01T10:00:00.000 and @2020-03-01
define D86: difference in days between @2020-03-01 and @2020-03-01T10:00:00.000
define D87: difference in days between @2020-03-01T10:00:00.000 and @2020-03-01
define D88: difference in hours between @2012-01-01 and @2012-01-02T05:00
define D89: hours between @2012-01-01 and @2012-01-02T05:00
define D90: days between @2012-01-01 and @2012-01-05T00:00:00.000
define D91: days between @2012-01-01 and @2012-01-05T10:00:00.000
define D92: duration in days of Interval[@2020-03-01, @2020-03-01T10:00:00.000]
define D93: duration in days of Interval[@2020-03-01T10:00:00.000, @2020-03-01]
define D94: Max({ @2020-03-01, @2020-03-01T10:00:00.000 })
define D95: Max({ @2020-03-01T10:00:00.000, @2020-03-01 })
define D96: Min({ @2020-03-01, @2020-03-01T10:00:00.000 })
define D97: Count({ @2020-03-01, @2020-03-01T10:00:00.000 })
define D98: Mode({ @2020-03-01, @2020-03-01T10:00:00.000 })
define D99: distinct { @2020-03-01, @2020-03-01T10:00:00.000 }
define D100: First({ @2020-03-01, @2020-03-01T10:00:00.000 })
define D101: @2020-03-01 in day of Interval[@2020-03-01T10:00:00.000, @2020-03-05T10:00:00.000]
define D103: Interval[@2020-03-01, @2020-03-05] overlaps day of Interval[@2020-03-01T10:00:00.000, @2020-03-05T10:00:00.000]
define D104: Interval[@2020-03-01T10:00:00.000, @2020-03-05T10:00:00.000] overlaps day of Interval[@2020-03-01, @2020-03-05]
define D105: F(@2020-03-01)
define D106: G(Interval[@2020-03-01, @2020-03-05])
define D107: F(@2020-03-01T10:00:00.000)
define D108: MP
define D109: P
define D110: Interval[@2020-03-01T10:00:00.000, @2020-03-05T10:00:00.000] during MP
define D111: { { @2020-03-01 }, { @2020-03-01T10:00:00.000 } }
define D112: { Interval[@2020-03-01, @2020-03-01], Interval[@2020-03-01T10:00:00.000, @2020-03-01T10:00:00.000] }
define D113: { Tuple { a: @2020-03-01 }, Tuple { a: @2020-03-01T10:00:00.000 } }
define D114: ({ @2020-03-01 }) X where X < @2020-03-01T10:00:00.000
define D115: ({ @2020-03-01 }) X return X = @2020-03-01T10:00:00.000
define D116: ({ @2020-03-01, @2020-03-01T10:00:00.000 }) X return X
define D117: ({ @2020-03-01 }) X let Y: @2020-03-01T10:00:00.000 return X < Y
define D118: ({ @2020-03-01T10:00:00.000 }) X where X > @2020-03-01
define D119: { @2020-03-01, null, @2020-03-01T10:00:00.000 }
define D120: Interval[@2020-03-01, null] contains @2020-03-01T10:00:00.000
define D121: Interval[null, @2020-03-01T10:00:00.000] contains @2020-03-01
define D122: { @2020-03-01, @2020-03-01T10:00:00.000 } includes @2020-03-01
define D123: @2020-03-01 in { @2020-03-01T10:00:00.000, null }
define D124: collapse { Interval[@2020-03-01, @2020-03-01], Interval[@2020-03-01T10:00:00.000, @2020-03-01T10:00:00.000] }
define D125: expand { Interval[@2020-03-01, @2020-03-01] } per day = { Interval[@2020-03-01T10:00:00.000, @2020-03-01T10:00:00.000] }
define D127: @2020-03-01 in Interval[@2020-03-01T10:00:00.000, null]
define D128: @2020-03-01 in Interval[null, @2020-03-01T10:00:00.000]
