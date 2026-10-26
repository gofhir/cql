library Differential version '1.0'
// Written by hand: the cases of docs/plans/008-fhirpath-1-16.md, section 4.
// CQL Reference (Appendix B), formatting strings: "ss - A full two digit second
// value (00..59)"; the range of DateTime ends at @9999-12-31T23:59:59.999 and
// that of Time at @T23:59:59.999.
define D0: @2016-12-31T23:59:60Z
define D1: @2016-12-31T23:59:60.5Z
define D2: ToDateTime('2016-12-31T23:59:60Z')
define D3: ToTime('23:59:60')
define D4: @T23:59:60
define D5: DateTime(2016, 12, 31, 23, 59, 60, 0, 0)
define D6: Time(23, 59, 60)
define D7: second from ToDateTime('2016-12-31T23:59:60Z')
define D8: @2016-12-31T23:59:59.999Z < @2017-01-01T00:00:00.000Z
define D9: ToTime('23:59:59.999')
define D10: ToDateTime('2016-12-31T23:60:00Z')
define D11: Time(23, 60, 0)
// Offsets, whole and partial, compared at every precision (CQL Reference,
// Equal, Less and the comparison of DateTime values).
define D12: @2020-01-01T10+05:30 = @2020-01-01T04Z
define D13: @2020-01-01T10+05:30 < @2020-01-01T05Z
define D14: @2020-01-01T10+05:30 > @2020-01-01T04Z
define D15: @2020-01-01T10+05:30 <= @2020-01-01T04Z
define D16: @2020-01-01T10+05:30 >= @2020-01-01T05Z
define D17: @2020-01-01T10:00+05:30 = @2020-01-01T04:30Z
define D18: @2020-01-01T10+05:00 = @2020-01-01T05Z
define D19: @2020-01-01T10+05:00 < @2020-01-01T06Z
define D20: @2020-01-01T10+05:45 > @2020-01-01T04Z
define D21: @2020-01-01T10:15+05:45 = @2020-01-01T04:30Z
define D22: @2020-01-01T10+05:30 same hour as @2020-01-01T04Z
define D23: @2020-01-01T10+05:30 same or before hour of @2020-01-01T05Z
define D24: Interval[@2020-01-01T04Z, @2020-01-01T06Z] contains @2020-01-01T10+05:30
define D25: @2020-01-01T10+05:30 in Interval[@2020-01-01T05Z, @2020-01-01T06Z]
define D26: @2020-01-01T10-03:30 = @2020-01-01T13:30Z
define D27: @2020-01-01T23+05:30 < @2020-01-01T18Z
define D28: minutes between @2020-01-01T10:00:00.000+05:30 and @2020-01-01T06:00:00.000Z
define D29: hour from @2020-01-01T10+05:30
define D30: ToDateTime('2020-01-01+05:00')
define D31: ToDateTime('2020-01-01T10+05:30')
define D32: @2020-01-01T10+05:30 ~ @2020-01-01T04Z
define D33: @2020-01-01T10:30+05:30 < @2020-01-01T05:01Z
// A DateTime to the hour against one to the minute or finer, at offsets that
// differ by other than whole hours: the hour of 10+05:30 runs from 04:30Z to
// 05:30Z, and the order is known only outside it (the decision above).
define D34: @2020-01-01T10+05:30 < @2020-01-01T05:00Z
define D35: @2020-01-01T10+05:30 = @2020-01-01T05:00Z
define D36: @2020-01-01T05:00Z > @2020-01-01T10+05:30
define D37: @2020-01-01T10+05:30 < @2020-01-01T05:00:00.000Z
define D38: @2020-01-01T10+05:30 > @2020-01-01T04:59Z
define D39: @2020-01-01T10+05:30 < @2020-01-01T06:00Z
define D40: @2020-01-01T10+05:30 > @2020-01-01T04:00Z
define D41: @2020-01-01T10+05:30 < @2020-01-01T05:30Z
define D42: @2020-01-01T05:29Z > @2020-01-01T10+05:30
define D47: @2020-01-01T10+05:30 > @2020-01-01T04:29:30Z
define D48: @2020-01-01T10+05:30 > @2020-01-01T04:29Z
// The leap second through the conversion operators: "ss - A full two digit
// second value (00..59)" (CQL Reference, formatting strings).
define D43: ConvertsToDateTime('2016-12-31T23:59:60Z')
define D44: convert '2016-12-31T23:59:60Z' to DateTime
define D45: convert '23:59:60' to Time
define D46: ConvertsToTime('23:59:60')
