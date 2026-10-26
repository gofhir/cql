library Differential version '1.0'
// Written by hand: the cases of docs/plans/008-fhirpath-1-16.md, section 1 and 4.
// CQL Reference (Appendix B), Add and Subtract: "The unit of the result will be
// the most granular unit of either input."
define D0: 2 'm' - 100 'cm'
define D1: 100 'cm' - 2 'm'
define D2: 100 'cm' + 2 'm'
define D3: 2 'm' + 100 'cm'
define D4: 1 'g' + 500 'mg'
define D5: 500 'mg' + 1 'g'
define D6: 3 'm' - 1 'km'
define D7: 1 'km' - 3 'm'
define D8: 1 'L' - 250 'mL'
define D9: 1 'h' + 30 'min'
define D10: 1.0 'm' + 1 'mm'
define D11: 1 '[in_i]' + 1 'cm'
define D12: 2 'kg' - 2000 'g'
define D13: 1.555555555 'g' + 1 'mg'
define D14: 1 'mg' + 1 'mg'
define D15: 5 'mg' + 1 '1'
define D16: 1 '1' + 1 '%'
define D17: 1 'Cel' + 1 'K'
define D18: 37 'Cel' + 1 'K'
// What is defined through Add and Subtract inherits their unit.
define D19: Sum({ 1 'g', 500 'mg' })
define D20: Sum({ 500 'mg', 1 'g' })
define D21: Avg({ 1 'g', 500 'mg' })
define D22: Median({ 1 'g', 500 'mg', 2000 'mg' })
define D23: Max({ 1 'g', 500 'mg' }) - Min({ 1 'g', 500 'mg' })
define D24: width of Interval[100 'cm', 2 'm']
define D25: Size(Interval[100 'cm', 2 'm'])
define D26: expand Interval[100 'cm', 2 'm'] per 50 'cm'
define D27: collapse { Interval[1 'g', 2 'g'], Interval[1500 'mg', 3000 'mg'] }
// CQL Reference, Successor and Predecessor: the step is the smallest the
// precision of the type has, in the quantity's own unit.
define D28: successor of (2 'm' - 100 'cm')
define D29: predecessor of (1 'g' + 1 'mg')
define D30: predecessor of (1 'g' - 500 'mg')
define D31: (1 'g' + 1 'mg').value
define D32: Interval(2 'm' - 100 'cm', 2 'm'] = Interval[1.00000001 'm', 2 'm']
define D33: (2 'm' - 100 'cm') = 1 'm'
// Author's Guide, Quantities: "Durations above days (and weeks) are calendar
// durations that are not comparable with definite quantity UCUM duration
// units"; a month or a year has no fixed number of days.
define D34: 1 day + 1 hour
define D35: 1 'd' + 1 'h'
define D36: 1 day + 1 'h'
define D37: 1 hour + 30 minutes
define D38: 1 week + 1 day
define D39: 1 month + 1 month
define D40: 1 year + 2 years
define D41: 1 month + 1 day
define D42: 1 year + 1 month
define D43: 1 year + 1 day
define D44: 1 month - 1 day
define D45: 1 month + 1 hour
define D46: 1 month + 1 'd'
define D47: 1 'mo' + 1 'd'
define D48: @2020-01-31 + (1 month + 1 day)
define D49: (@2020-01-31 + 1 month) + 1 day
define D50: @2020-01-31T00:00:00Z + (1 year + 1 month) = @2021-02-28T00:00:00Z
define D51: @2020-01-31 - (1 month + 1 day)
// The same calendar durations through what is defined by addition: a week is
// below the month, Sum adds its elements ("Sum(X) = X1 + X2 + ...", CQL
// Reference, Sum), and Width is "(end of argument – start of argument)" (CQL
// Reference, Width).
define D52: 1 month + 1 week
define D53: 2 years - 1 week
define D54: Sum({ 1 month, 1 day })
define D55: Sum({ 1 day, 1 month })
define D56: width of Interval[1 day, 1 month]
define D57: Sum({ 1 year, 1 month })
// Each calendar duration keyword, singular and plural, on either side of the
// month (Author's Guide, Quantities, as above).
define D58: 2 months + 1 day
define D59: 1 month + 30 minutes
define D60: 1 minute + 1 year
define D61: 1 month + 1 second
define D62: 2 seconds - 1 year
define D63: 1 month + 1 millisecond
define D64: 3 milliseconds + 2 months
define D65: 2 years - 1 hour
define D66: 1 month + 2 hours
define D67: 1 year + 2 days
define D68: 1 month + 2 weeks
// The spread of quantities in two units: "the statistical standard deviation"
// and "the statistical variance" (CQL Reference, StdDev and Variance) are
// taken over the deviations, each a difference, in "the most granular unit of
// either input" (Add and Subtract).
define D69: PopulationStdDev({ 1 'g', 500 'mg' })
define D70: PopulationStdDev({ 500 'mg', 1 'g' })
define D71: StdDev({ 1 'g', 500 'mg' })
define D72: Variance({ 1 'g', 500 'mg' })
