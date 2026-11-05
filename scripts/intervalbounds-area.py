#!/usr/bin/env python3
# Writes testdata/snapshots/intervalbounds.cql, the area of plan 023
# (docs/plans/023-interval-start-end.md), to standard output, and with --oracle
# the answers its independent oracle gives, to standard output as well.
# Usage: python3 scripts/intervalbounds-area.py > testdata/snapshots/intervalbounds.cql
#        python3 scripts/intervalbounds-area.py --oracle > testdata/intervalbounds/oracle.tsv
#
# The oracle is pure Python and shares nothing with the engine: Start and End by
# CQL 1.5.3 (https://cql.hl7.org/N1A/, Appendix B), the successor or predecessor
# of an open boundary at its own precision, the minimum or maximum of the point
# type for a closed null, null for an open null; days and months counted with the
# Julian day number, hours to milliseconds by instants in UTC, with the extremes
# of a DateTime at UTC, as Java. It answers the rows where both ends are known
# and at the unit's precision or finer; the rest it leaves to the references.
import sys

EVAL_OFFSET = -3 * 60  # minutes: the evaluation offset of the snapshots

DATE_UNITS = ["years", "months", "weeks", "days"]
TIME_UNITS = ["hours", "minutes", "seconds", "milliseconds"]
UNIT_PRECISION = {"years": 1, "months": 2, "weeks": 3, "days": 3, "hours": 4, "minutes": 5, "seconds": 6, "milliseconds": 7}
TIME_PRECISION = {"hours": 1, "minutes": 2, "seconds": 3, "milliseconds": 4}


def dim(y, m):
    if m == 2:
        return 29 if (y % 4 == 0 and y % 100 != 0) or y % 400 == 0 else 28
    return 30 if m in (4, 6, 9, 11) else 31


class V:
    def __init__(self, kind, fields, off=""):
        self.kind, self.f, self.off = kind, list(fields), off

    @property
    def prec(self):
        return len(self.f)

    def bounds(self, f, i):
        if self.kind == "Time":
            return 0, (23, 59, 59, 999)[i]
        if i == 0:
            return 1, 9999
        if i == 1:
            return 1, 12
        if i == 2:
            return 1, dim(f[0], f[1])
        return 0, (23, 59, 59, 999)[i - 3]

    def step(self, f, i, d):
        f = list(f)
        f[i] += d
        if i == 0:
            lo, hi = self.bounds(f, 0)
            return f if lo <= f[0] <= hi else None
        lo, hi = self.bounds(f, i)
        if f[i] > hi:
            f[i] = lo
            return self.step(f, i - 1, 1)
        if f[i] < lo:
            g = self.step(f, i - 1, -1)
            if g is None:
                return None
            g[i] = self.bounds(g, i)[1]
            return g
        return f

    def moved(self, d):
        f = self.step(self.f, len(self.f) - 1, d)
        return None if f is None else V(self.kind, f, self.off)

    def text(self):
        f = self.f
        if self.kind == "Time":
            return ":".join("%02d" % x for x in f[:3]) + (".%03d" % f[3] if len(f) == 4 else "")
        s = "%04d" % f[0]
        if len(f) > 1:
            s += "-%02d" % f[1]
        if len(f) > 2:
            s += "-%02d" % f[2]
        if self.kind == "Date":
            return s
        if len(f) > 3:
            s += "T%02d" % f[3]
        if len(f) > 4:
            s += ":%02d" % f[4]
        if len(f) > 5:
            s += ":%02d" % f[5]
        if len(f) > 6:
            s += ".%03d" % f[6]
        return s + self.off

    def literal(self):
        if self.kind == "Date":
            return "@" + self.text()
        if self.kind == "Time":
            return "@T" + self.text()
        return "@" + self.text() + ("T" if len(self.f) <= 3 else "")

    def answer(self):
        return f"{self.kind} {self.text()}"

    def instant(self):
        """Milliseconds from 1970-01-01T00:00:00Z (from midnight for a Time)."""
        f = self.f
        if self.kind == "Time":
            g = f + [0] * (4 - len(f))
            return ((g[0] * 60 + g[1]) * 60 + g[2]) * 1000 + g[3]
        g = self.padded()
        ms = (julian(g[0], g[1], g[2]) - 2440588) * 86400000
        ms += ((g[3] * 60 + g[4]) * 60 + g[5]) * 1000 + g[6]
        if self.kind == "Date":
            return ms
        if self.off == "Z":
            return ms
        if self.off:
            sign = -1 if self.off[0] == "-" else 1
            return ms - sign * (int(self.off[1:3]) * 60 + int(self.off[4:6])) * 60000
        return ms - EVAL_OFFSET * 60000

    def padded(self):
        """The seven fields of a Date or DateTime, a missing month or day as 1 and the rest as 0."""
        return self.f + [1] * max(0, 3 - len(self.f)) + [0] * max(0, 7 - max(3, len(self.f)))

    def wall(self):
        """The clock of the evaluation's offset, as the boundaries of a unit read it."""
        return self.instant() if self.kind in ("Time", "Date") else self.instant() + EVAL_OFFSET * 60000


def julian(y, m, d):
    a = (14 - m) // 12
    yy = y + 4800 - a
    mm = m + 12 * a - 3
    return d + (153 * mm + 2) // 5 + 365 * yy + yy // 4 - yy // 100 + yy // 400 - 32045


def D(*f):
    return V("Date", f)


def DT(*f, off=""):
    return V("DateTime", f, off)


def T(*f):
    return V("Time", f)


EXTREMES = {
    "Date": (D(1, 1, 1), D(9999, 12, 31)),
    "DateTime": (DT(1, 1, 1, 0, 0, 0, 0, off="Z"), DT(9999, 12, 31, 23, 59, 59, 999, off="Z")),
    "Time": (T(0, 0, 0, 0), T(23, 59, 59, 999)),
}


class Interval:
    """lo and hi are a V or None (null); a closed flag each; kind is the point type."""

    def __init__(self, kind, lo, lc, hi, hc):
        self.kind, self.lo, self.lc, self.hi, self.hc = kind, lo, lc, hi, hc

    def cql(self):
        low = "null" if self.lo is None else self.lo.literal()
        high = "null" if self.hi is None else self.hi.literal()
        if self.lo is None and self.hi is None:
            low = f"null as {self.kind}"
        return ("[" if self.lc else "(") + low + ", " + high + ("]" if self.hc else ")")

    def start(self):
        if self.lo is None:
            return EXTREMES[self.kind][0] if self.lc and self.hi is not None else None
        return self.lo if self.lc else self.lo.moved(1)

    def end(self):
        if self.hi is None:
            return EXTREMES[self.kind][1] if self.hc and self.lo is not None else None
        return self.hi if self.hc else self.hi.moved(-1)


def trunc_div(a, b):
    q = abs(a) // b
    return q if a >= 0 else -q


def count(unit, a, b, difference):
    """Whole periods (a duration) or boundaries crossed (a difference) from a to b."""
    if unit in ("years", "months"):
        fa, fb = a.padded(), b.padded()
        months = (fb[0] - fa[0]) * 12 + fb[1] - fa[1]
        if difference:
            return fb[0] - fa[0] if unit == "years" else months

        def later(x, y):
            for i in range(2, 7):
                if x[i] != y[i]:
                    return x[i] > y[i]
            return False
        if months > 0 and later(fa, fb):
            months -= 1
        elif months < 0 and later(fb, fa):
            months += 1
        return trunc_div(months, 12) if unit == "years" else months
    length = {"weeks": 7 * 86400000, "days": 86400000, "hours": 3600000, "minutes": 60000, "seconds": 1000, "milliseconds": 1}[unit]
    if not difference:
        if unit in ("weeks", "days") and a.kind == "DateTime" and a.off != b.off:
            # CQL 1.5.3, Appendix B, Duration: the offsets are normalized "only
            # when the comparison precision is hours, minutes, seconds, or
            # milliseconds"; in days the dates count as written, left to the
            # references (backlog 1c28).
            return None
        return trunc_div(b.instant() - a.instant(), length)
    if unit == "weeks":
        def sunday(v):
            jd = julian(*v.padded()[:3])
            return jd - (jd + 1) % 7
        return (sunday(b) - sunday(a)) // 7
    if unit == "days":
        def day(v):
            return julian(*v.padded()[:3])
        if a.kind == "DateTime" and a.off != b.off:
            return None  # the day as written: the offsets differ, left to the references
        return day(b) - day(a)
    return b.wall() // length - a.wall() // length


def integer(n):
    if n is None:
        return None
    if n >= 1 << 31 or n < -(1 << 31):
        return "null"
    return f"Integer {n}"


def unit_precision(kind, unit):
    return TIME_PRECISION[unit] if kind == "Time" else UNIT_PRECISION[unit]


def measure(iv, unit, difference):
    """The oracle's answer, or None where it leaves the row to the references."""
    a, b = iv.start(), iv.end()
    if a is None or b is None:
        return "null"
    need = unit_precision(iv.kind, unit)
    if a.prec < need or b.prec < need:
        return None
    # A duration counts whole periods, and an end at the unit's own precision
    # leaves the finer fields unknown: an uncertainty (plan 016), left to the
    # references. Days and weeks of a Date, and seconds and milliseconds, have none.
    if not difference and need in (a.prec, b.prec) and (
            (iv.kind == "Date" and need in (1, 2)) or (iv.kind == "DateTime" and need in (1, 2, 3, 4, 5))
            or (iv.kind == "Time" and need in (1, 2))):
        return None
    if iv.kind == "DateTime" and a.off != b.off and unit in ("years", "months", "weeks", "days") and a.prec != b.prec:
        return None
    if a.prec != b.prec and unit in ("years", "months") and need < a.prec and need < b.prec:
        return None
    n = count(unit, a, b, difference)
    return integer(n)


cases = []  # (expression, oracle answer or None)


def add_interval(iv):
    units = {"Date": DATE_UNITS, "DateTime": DATE_UNITS + TIME_UNITS, "Time": TIME_UNITS}[iv.kind]
    text = "Interval" + iv.cql()
    for u in units:
        cases.append((f"duration in {u} of {text}", measure(iv, u, False)))
        cases.append((f"difference in {u} of {text}", measure(iv, u, True)))
    for word, point in (("start", iv.start()), ("end", iv.end())):
        cases.append((f"{word} of {text}", "null" if point is None else point.answer()))


def forms(kind, lo, hi, which):
    """Intervals over lo and hi: which lists (low closed, high closed), a bound
    of None being a null written in place of the value."""
    for lc, hc, lo_null, hi_null in which:
        yield Interval(kind, None if lo_null else lo, lc, None if hi_null else hi, hc)


ALL4 = [(True, True, False, False), (False, False, False, False), (True, False, False, False), (False, True, False, False)]
NULLS = [(True, True, False, True), (False, True, False, True), (True, True, True, False), (True, False, True, False),
         (True, False, False, True), (False, True, True, False)]

A, B, C = D(2012, 1, 1), D(2012, 1, 5), D(2012, 1, 2)
for iv in forms("Date", A, B, ALL4):
    add_interval(iv)
for iv in forms("Date", A, C, [(False, False, False, False), (True, False, False, False)]):
    add_interval(iv)  # no points, and one point
for iv in forms("Date", A, A, NULLS[:4]):
    add_interval(iv)
MIN, MAX = D(1, 1, 1), D(9999, 12, 31)
for iv in forms("Date", MIN, MAX, [(True, True, False, False), (False, False, False, False), (True, True, False, True), (True, True, True, False)]):
    add_interval(iv)
add_interval(Interval("Date", None, True, MIN, False))  # no predecessor of the minimum
add_interval(Interval("Date", MAX, False, None, True))  # no successor of the maximum
add_interval(Interval("Date", None, True, None, True))
add_interval(Interval("Date", D(2012, 3), True, D(2012, 9), False))
add_interval(Interval("Date", D(2012, 3), False, None, True))
add_interval(Interval("Date", D(2012), True, D(2015), True))
add_interval(Interval("Date", D(9999, 12), False, None, True))
add_interval(Interval("Date", None, True, D(1, 1), False))

TDEF = DT(2012, 1, 1, 0, 0, 0, 0)
T1 = DT(2012, 1, 5, 10, 0, 0, 0)
H, H2 = DT(2012, 1, 1, 10, 0), DT(2012, 1, 1, 12, 0)
S, S2 = DT(2012, 1, 1, 10, 0, 0, 0), DT(2012, 1, 1, 10, 0, 1, 0)
add_interval(Interval("DateTime", H, False, H2, False))
add_interval(Interval("DateTime", H, True, H2, True))
add_interval(Interval("DateTime", S, False, S2, False))
for iv in forms("DateTime", TDEF, T1, [(True, True, False, False), (False, False, False, False)]):
    add_interval(iv)
add_interval(Interval("DateTime", None, False, TDEF, True))
add_interval(Interval("DateTime", TDEF, True, None, False))
Z0, Z1 = DT(2012, 1, 1, 0, 0, 0, 0, off="Z"), DT(2012, 1, 5, 10, 0, 0, 0, off="Z")
add_interval(Interval("DateTime", Z0, True, Z1, False))
P0 = DT(2012, 1, 1, 0, 0, 0, 0, off="+03:00")
add_interval(Interval("DateTime", P0, True, None, True))
add_interval(Interval("DateTime", None, True, DT(2012, 1, 5, 10, 0, 0, 0, off="+03:00"), False))
add_interval(Interval("DateTime", DT(9999, 12, 31, 0, 0, 0, 0), True, None, True))
add_interval(Interval("DateTime", DT(9999, 12, 31, 23, 59, 59, 999), False, None, True))
add_interval(Interval("DateTime", None, True, DT(1, 1, 1, 0, 0, 0, 0), False))
add_interval(Interval("DateTime", DT(2012, 1, 1, 10), True, DT(2012, 1, 5, 11), True))
add_interval(Interval("DateTime", DT(2012), False, DT(2013), False))

for iv in forms("Time", T(10, 0), T(12, 0), ALL4[:3]):
    add_interval(iv)
for iv in forms("Time", T(10, 0), T(12, 0), NULLS[1:4]):
    add_interval(iv)
add_interval(Interval("Time", T(10), True, T(11), True))
add_interval(Interval("Time", T(23, 59, 59, 999), False, None, True))
add_interval(Interval("Time", None, True, T(0, 0, 0, 0), False))
add_interval(Interval("Time", T(0, 0, 0, 0), True, T(23, 59, 59, 999), True))

# Controls, answered by the references alone: an interval promoted from Date to
# DateTime by a conditional (plan 019), one that reaches a duration through a
# query, and an uncertain duration. A FHIR Period is in engine_intervalbounds_test.go:
# the harness gives the reference engines no FHIR model for this area.
cases += [
    ("duration in hours of (if true then Interval[@9999-12-30, null] else Interval[@9999-12-30T00:00:00.000, null])", None),
    ("duration in days of (if true then Interval[@2012-01-01, @2012-01-05] else Interval[@2012-01-01T00:00:00.000, @2012-01-05T00:00:00.000])", None),
    ("({Interval[@2012-01-01, null], Interval(@2012-01-01, @2012-01-05)}) I return duration in days of I", None),
    ("({Interval[@2012-01-01, null], Interval(@2012-01-01, @2012-01-05)}) I return end of I", None),
    ("duration in days of Interval[@2012-01, @2012-05-01]", None),
    ("duration in hours of Interval[@2012-01-01T10, @2012-01-01T12:30]", None),
    ("duration in days of Interval[null as Date, null]", "null"),
]

if "--oracle" in sys.argv:
    for i, (_, answer) in enumerate(cases):
        if answer is not None:
            print(f"D{i}\t{answer}")
    sys.exit(0)

print("""library Differential version '1.0'
using FHIR version '4.0.1'
include FHIRHelpers version '4.0.1'
// Generated by scripts/intervalbounds-area.py: the cases of
// docs/plans/023-interval-start-end.md. `duration in X of I` and `difference in
// X of I` measure between `start of I` and `end of I` (CQL 1.5.3, Author's
// Guide, Interval Operators), and those read a closed null boundary as the
// minimum or maximum of the point type (Appendix B, Start and End). Date, DateTime
// (with no offset, Z and +03:00) and Time; every form of a boundary; the edges of
// the calendar and of the clock; then controls that only the references answer.""")
for i, (expr, _) in enumerate(cases):
    print(f"define D{i}: {expr}")
