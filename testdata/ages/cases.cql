library Ages version '1.0'

using FHIR version '4.0.1'

include FHIRHelpers version '4.0.1'

parameter "Measurement Period" Interval<DateTime>

context Patient

define AgeInYears: AgeInYears()
define AgeInMonths: AgeInMonths()
define AgeInWeeks: AgeInWeeks()
define AgeInDays: AgeInDays()
define AgeInHours: AgeInHours()
define AgeInMinutes: AgeInMinutes()
define AgeInSeconds: AgeInSeconds()
define AgeInYearsAtStart: AgeInYearsAt(start of "Measurement Period")
define AgeInMonthsAtStart: AgeInMonthsAt(start of "Measurement Period")
define AgeInWeeksAtStart: AgeInWeeksAt(start of "Measurement Period")
define AgeInDaysAtStart: AgeInDaysAt(start of "Measurement Period")
define AgeInHoursAtStart: AgeInHoursAt(start of "Measurement Period")
define AgeInMinutesAtStart: AgeInMinutesAt(start of "Measurement Period")
define AgeInSecondsAtStart: AgeInSecondsAt(start of "Measurement Period")
define AgeInYearsAtDate: AgeInYearsAt(@2019-01-01)
define AgeInMonthsAtDate: AgeInMonthsAt(@2019-01-01)

define CD3: CalculateAgeInWeeksAt(@2019-01, @2019-03-01)
define CD4: CalculateAgeInDaysAt(@2019-01, @2019-03-01)
define CD18: CalculateAgeInMonthsAt(@1965, @2019-01-01)
define CD6: CalculateAgeInDaysAt(@2019-01-01T10:00, @2019-01-02T09:00)
define CD8: CalculateAgeInHoursAt(@2019-01-01T10:00, @2019-01-02T09:00)
define CD15: CalculateAgeInMinutesAt(@2019-01-01T10:00, @2019-01-02T09:00)
define CD16: CalculateAgeInSecondsAt(@2019-01-01T10:00, @2019-01-01T10:01)
define CH5: CalculateAgeInHoursAt(@2019-01-01, @2019-01-02)
define CNegative: CalculateAgeInYearsAt(@2014-02-28, @2012-02-29)
define CZulu: CalculateAgeInYearsAt(@2000-01-01, @2026-01-01T00:00:00.000Z)
define CNormative: CalculateAgeInYearsAt(@2000-01-01, @2015-01-01)
define CLeapDay: CalculateAgeInYearsAt(@2000-02-29, @2021-02-28)
define CNullAsOf: CalculateAgeInYearsAt(@2000-01-01, null)
define CYearsToday: CalculateAgeInYears(@1965-06-01)
define CMonthsToday: CalculateAgeInMonths(@1965-06-01)
define CDaysToday: CalculateAgeInDays(@2025-12-01)
define CDaysNow: CalculateAgeInDays(@2025-12-01T00:00:00.000)
define CHoursNow: CalculateAgeInHours(@2025-12-31T00:00:00.000)
define CMinutesNow: CalculateAgeInMinutes(@2025-12-31T23:00:00.000)
define CSecondsNow: CalculateAgeInSeconds(@2025-12-31T23:59:00.000)
define CWeeksAtDate: CalculateAgeInWeeksAt(@2012-02-29, @2012-03-07)
