#!/usr/bin/env python3
# Writes testdata/snapshots/durationrange.cql, the area of plan 021
# (docs/plans/021-duration-integer-range.md) and of plan 022
# (docs/plans/022-duration-out-of-range.md), to standard output.
# Usage: python3 scripts/durationrange-area.py > testdata/snapshots/durationrange.cql

# Each pair is (low, high) as full instants; every pair is written as Date and
# as DateTime, with and without a stated offset, in every unit valid for it.
pairs = [
    ("0001-01-01T00:00:00.000", "0001-01-05T00:00:00.000"),
    ("0001-01-01T00:00:00.000", "0002-01-01T00:00:00.000"),
    ("0001-01-01T00:00:00.000", "2020-01-01T00:00:00.000"),
    ("0001-01-01T00:00:00.000", "9999-12-31T00:00:00.000"),
    ("0001-01-02T00:00:00.000", "9999-12-31T23:59:59.999"),
    ("1730-01-01T00:00:00.000", "2020-01-01T00:00:00.000"),
    ("1727-01-01T00:00:00.000", "2020-01-01T00:00:00.000"),
    ("1727-06-15T12:30:00.000", "2020-03-01T08:15:00.000"),
    ("2012-01-01T00:00:00.000", "2012-01-25T20:31:23.647"),
]
date_units = ["years", "months", "weeks", "days"]
time_units = ["hours", "minutes", "seconds", "milliseconds"]
offsets = ["", "Z", "+03:00"]

out = []


def dt(v, off):
    return "@" + v + off


for low, high in pairs:
    for a, b in ((low, high), (high, low)):
        for off in offsets:
            x, y = dt(a, off), dt(b, off)
            for u in date_units + time_units:
                out.append(f"{u} between {x} and {y}")
                out.append(f"difference in {u} between {x} and {y}")
            if (a, b) == (low, high):
                for u in date_units + time_units:
                    out.append(f"duration in {u} of Interval[{x}, {y}]")
                    out.append(f"difference in {u} of Interval[{x}, {y}]")
                for u in ["Years", "Months", "Weeks", "Days", "Hours", "Minutes", "Seconds"]:
                    out.append(f"CalculateAgeIn{u}At({x}, {y})")
        da, db = "@" + a[:10], "@" + b[:10]
        for u in date_units:
            out.append(f"{u} between {da} and {db}")
            out.append(f"difference in {u} between {da} and {db}")
        if (a, b) == (low, high):
            for u in ["Years", "Months", "Weeks", "Days"]:
                out.append(f"CalculateAgeIn{u}At({da}, {db})")
# Year 1 with an offset that puts the instant on the calendar's first hour, and
# imprecise operands whose range starts at the first instant.
out += [
    "hours between @0001-01-01T03:00:00.000+03:00 and @0001-01-02T00:00:00.000Z",
    "days between @0001-01-01T03:00:00.000+03:00 and @0001-01-05T00:00:00.000Z",
    "days between @0001 and @0002",
    "days between @0001-01 and @0001-03",
    "days between @0001-01-01T and @0001-01-05T00:00:00.000",
    "hours between @0001-01-01T and @0001-01-02T00:00:00.000Z",
    "days between @0001 and @9999",
    "hours between @1727 and @2020-01-01T00:00:00.000",
    "difference in days between @0001 and @0002",
    "difference in hours between @0001-01-01T and @0001-01-02T00:00:00.000Z",
]
# Plan 022: the edge of an Integer. A duration, a difference and an age that
# count milliseconds, seconds or minutes can pass 2^31-1 within the calendar. Each
# base is followed by ends that are 2^31-2, 2^31-1 and 2^31 units away (and a
# fraction of a unit, which is dropped toward zero), in both orders (-2^31 is
# representable), as DateTime with no offset, Z and +03:00. The last ends are
# counted by hand: 2^31 ms is 24 days 20:31:23.648, 2^31 s is 24855 days 03:14:08
# and 2^31 min is 1491308 days 02:08.
edges = [
    ("milliseconds", "2012-01-01T00:00:00.000",
     ["2012-01-25T20:31:23.646", "2012-01-25T20:31:23.647", "2012-01-25T20:31:23.648", "2012-01-25T20:31:23.649"]),
    ("seconds", "2000-01-01T00:00:00.000",
     ["2068-01-19T03:14:06.000", "2068-01-19T03:14:07.000", "2068-01-19T03:14:07.999", "2068-01-19T03:14:08.000",
      "2068-01-19T03:14:09.000"]),
    ("minutes", "0001-01-01T00:00:00.000",
     ["4084-01-24T02:06:00.000", "4084-01-24T02:07:00.000", "4084-01-24T02:07:59.999", "4084-01-24T02:08:00.000",
      "4084-01-24T02:09:00.000"]),
]
for unit, base, ends in edges:
    for end in ends:
        for off in offsets:
            x, y = dt(base, off), dt(end, off)
            out.append(f"{unit} between {x} and {y}")
            out.append(f"difference in {unit} between {x} and {y}")
            out.append(f"{unit} between {y} and {x}")
            out.append(f"difference in {unit} between {y} and {x}")
        for off in ["", "Z"]:
            x, y = dt(base, off), dt(end, off)
            out.append(f"duration in {unit} of Interval[{x}, {y}]")
            out.append(f"difference in {unit} of Interval[{x}, {y}]")
        if unit != "milliseconds":
            age = "Seconds" if unit == "seconds" else "Minutes"
            for off in ["", "Z"]:
                out.append(f"CalculateAgeIn{age}At({dt(base, off)}, {dt(end, off)})")
# An uncertainty inside the range, one that crosses the edge and one wholly out:
# null as a whole where a bound is no Integer (decision D1 of plan 022).
for a, b in [
    ("2012-01-01T00:00:00.000Z", "2012-01-25T20:29Z"),
    ("2012-01-01T00:00:00.000Z", "2012-01-25T20:31Z"),
    ("2012-01-01T00:00:00.000Z", "2012-01-26T00:00Z"),
    ("2000-01-01T00:00:00.000Z", "2068-01-19T03:12Z"),
    ("2000-01-01T00:00:00.000Z", "2068-01-19T03:14Z"),
    ("2000-01-01T00:00:00.000Z", "2068-01-20T00:00Z"),
]:
    unit = "milliseconds" if a.startswith("2012") else "seconds"
    for x, y in ((a, b), (b, a)):
        out.append(f"{unit} between @{x} and @{y}")
        out.append(f"difference in {unit} between @{x} and @{y}")
out += [
    "difference in milliseconds between @2012-01T and @2012-01-01T00:00:00.000",
    "difference in milliseconds between @2012-01T and @2012-01-01T",
    "milliseconds between @2012-01T and @2012-02-25T00:00:00.000",
    "seconds between @1950-01-01T00:00:00.000 and @2020-01-01T00:00:00.000",
    "difference in milliseconds between @2012-01-01T00:00:00.000 and @2012-02-01T00:00:00.000",
]
print("""library Differential version '1.0'
using FHIR version '4.0.1'
include FHIRHelpers version '4.0.1'
// Generated by scripts/durationrange-area.py: the cases of
// docs/plans/021-duration-integer-range.md. A duration, a difference and an age
// are measured over the whole calendar, from the year 1 to 9999: the first
// instant of the year 1 is a value like any other, and a span of more than 292
// years (the range of a Go time.Duration) is counted whole. Nine pairs of
// instants, both orders, as DateTime with no offset, Z and +03:00 and as Date, in
// every unit valid for them; then the year 1 at an offset and imprecise operands;
// then the edge of an Integer (docs/plans/022-duration-out-of-range.md), where a
// count past 2^31-1 or below -2^31 is null.""")
for i, e in enumerate(out):
    print(f"define D{i}: {e}")
