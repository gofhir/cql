#!/usr/bin/env python3
# Writes testdata/snapshots/mixeddates.cql, the area of plan 019
# (docs/plans/019-date-promotion.md), to standard output; with --oracle the
# answers of its independent oracle; with --pairs the implicit/explicit pairs of
# definitions; with --cases the definitions of the rows the reference translator
# converts, as a library of their own (testdata/mixeddates/cases.cql).
# Usage: python3 scripts/mixeddates-area.py > testdata/snapshots/mixeddates.cql
#        python3 scripts/mixeddates-area.py --oracle > testdata/mixeddates/oracle.tsv
#        python3 scripts/mixeddates-area.py --pairs > testdata/mixeddates/pairs.tsv
#        python3 scripts/mixeddates-area.py --cases > testdata/mixeddates/cases.cql
#
# A Date next to a DateTime is converted to a DateTime (CQL 1.5.3,
# https://cql.hl7.org/N1A/, Developer's Guide, Implicit Conversions, table 3-J:
# "ToDateTime(Date) Converts a Date value to a DateTime with all time components
# unspecified and the timezone offset of the request"). The area crosses the
# shapes of a Date (year, month, day), those of a DateTime (year to millisecond,
# with no offset, Z, +14:00 and -12:00), the operators and the order of the
# operands; the rows of the probe of the plan come first, written out as the
# translator was measured on them, then the grid, then the same rows with the
# ToDateTime written out, which the references answer as the implicit ones.
#
# The oracle is pure Python and shares nothing with the engine. It reads the
# comparison of two DateTimes as the specification words it (Comparison
# Operators, and Appendix B, Equal, Before, SameAs, Duration and Difference): the
# fields are compared in order up to the lesser precision, and where the values
# agree to there and one is the more precise the answer is null; offsets are
# normalized only when the comparison is made at the hour or finer, and a Date
# converted is at the evaluation's offset, -03:00. It answers the rows whose
# every step it can decide, and leaves the others to the references; it does not
# answer where the answer would rest on how a day-precision DateTime at an offset
# other than the evaluation's is read, which is plan 018's.
import sys

EVAL_OFFSET = -180  # minutes: the evaluation offset of the snapshots, -03:00

OFFSETS = {None: None, "Z": 0, "+14:00": 840, "-12:00": -720}


def julian(y, m, d):
    a = (14 - m) // 12
    yy = y + 4800 - a
    mm = m + 12 * a - 3
    return d + (153 * mm + 2) // 5 + 365 * yy + yy // 4 - yy // 100 + yy // 400 - 32045


def civil(jd):
    a = jd + 32044
    b = (4 * a + 3) // 146097
    c = a - 146097 * b // 4
    d = (4 * c + 3) // 1461
    e = c - 1461 * d // 4
    m = (5 * e + 2) // 153
    return 100 * b + d - 4800 + m // 10, m + 3 - 12 * (m // 10), e - (153 * m + 2) // 5 + 1


def dim(y, m):
    if m == 2:
        return 29 if (y % 4 == 0 and y % 100 != 0) or y % 400 == 0 else 28
    return 30 if m in (4, 6, 9, 11) else 31


class V:
    """A Date (kind "Date") or DateTime: its fields to the precision written,
    and the offset it was written with (None: none)."""

    def __init__(self, kind, fields, off=None):
        self.kind, self.f, self.off = kind, list(fields), off

    @property
    def prec(self):
        return len(self.f)

    def literal(self):
        f = self.f
        s = "%04d" % f[0]
        if len(f) > 1:
            s += "-%02d" % f[1]
        if len(f) > 2:
            s += "-%02d" % f[2]
        if self.kind == "DateTime":
            if len(f) <= 3:
                s += "T"
            else:
                s += "T%02d" % f[3]
                if len(f) > 4:
                    s += ":%02d" % f[4]
                if len(f) > 5:
                    s += ":%02d" % f[5]
                if len(f) > 6:
                    s += ".%03d" % f[6]
                s += self.off or ""
        return "@" + s

    def offset(self):
        return EVAL_OFFSET if self.off is None else OFFSETS[self.off]

    def filled(self, high):
        """The seven fields, the missing ones at their minimum or maximum."""
        f = list(self.f)
        y = f[0]
        m = f[1] if len(f) > 1 else (12 if high else 1)
        d = f[2] if len(f) > 2 else (dim(y, m) if high else 1)
        rest = [(23, 59, 59, 999) if high else (0, 0, 0, 0)][0]
        t = [f[3 + i] if len(f) > 3 + i else rest[i] for i in range(4)]
        return [y, m, d] + t

    def wall(self, high):
        """Milliseconds from 1970-01-01T00:00:00 on the clock of the value's own offset."""
        y, m, d, h, mi, s, ms = self.filled(high)
        return ((julian(y, m, d) - 2440588) * 24 + h) * 3600000 + (mi * 60 + s) * 1000 + ms

    def instant(self, high):
        return self.wall(high) - self.offset() * 60000

    def utc_fields(self):
        """The fields at UTC, to the precision of the value (it has an hour at least)."""
        ms = self.instant(False)
        days, rem = divmod(ms, 86400000)
        y, mo, d = civil(days + 2440588)
        full = [y, mo, d, rem // 3600000, rem // 60000 % 60, rem // 1000 % 60, rem % 1000]
        return full[:self.prec]


def D(*f):
    return V("Date", f)


def DT(*f, off=None):
    return V("DateTime", f, off)


class Leave(Exception):
    """The oracle does not decide this row."""


def cmp(a, b, cap=7):
    """-1, 0, 1 or None (uncertain) comparing two values at the lesser of their
    precisions and cap; raises Leave where an offset the evaluation does not
    have would decide a day or a coarser comparison."""
    p = min(a.prec, b.prec, cap)
    if p >= 4:
        fa, fb = a.utc_fields(), b.utc_fields()
    else:
        for v in (a, b):
            if v.prec >= 4 and v.offset() != EVAL_OFFSET:
                raise Leave()
        fa, fb = a.f, b.f
    for i in range(p):
        if fa[i] != fb[i]:
            return -1 if fa[i] < fb[i] else 1
    if p == cap or a.prec == b.prec:
        return 0
    return None


def and3(x, y):
    if x is False or y is False:
        return False
    if x is None or y is None:
        return None
    return True


def not3(x):
    return None if x is None else not x


def rel(a, b, op, cap=7):
    c = cmp(a, b, cap)
    if c is None:
        return None
    return {"=": c == 0, "<": c < 0, "<=": c <= 0, ">": c > 0, ">=": c >= 0}[op]


def eq3(a, b, cap=7):
    return rel(a, b, "=", cap)


def answer(x):
    if x is None:
        return "null"
    return "Boolean " + ("true" if x else "false")


def integer(n):
    return f"Integer {n}"


def coordinate(v, unit, high):
    """The index of the boundary period of the unit the value is in, on the clock of the evaluation."""
    y, m, d, h, mi, s, ms = v.filled(high)
    if unit == "years":
        return y
    if unit == "months":
        return y * 12 + m - 1
    if unit == "days":
        return julian(y, m, d)
    return v.instant(high) // 3600000


def measure(a, b, unit, form):
    """The oracle's answer to `unit between a and b`, in the form "difference"
    (boundaries crossed) or "duration" (whole periods), or Leave. Where a value
    is coarser than the unit it could be any of the periods it spans, and the
    answer is the range of what that gives (Appendix B, Duration: `days between
    @2017-08-07T17:00 and @2017-08-14T` is [6, 7])."""
    if form == "duration" and unit in ("years", "months"):
        raise Leave()  # calendar months with an uncertain day: left to the references
    if unit == "days" and (a.prec >= 4 or b.prec >= 4) and a.offset() != b.offset():
        raise Leave()  # the day as written, at offsets that differ
    if unit in ("days", "hours") and 1 in (a.prec, b.prec):
        # A year spans every offset's day boundaries, and the two references give
        # ranges that differ at the ends by an offset's worth: left to them.
        raise Leave()
    if form == "difference":
        lows = [coordinate(a, unit, False), coordinate(a, unit, True)]
        highs = [coordinate(b, unit, False), coordinate(b, unit, True)]
        lo, hi = highs[0] - lows[1], highs[1] - lows[0]
    else:
        length = {"days": 86400000, "hours": 3600000}[unit]

        def trunc(x):
            q = abs(x) // length
            return q if x >= 0 else -q
        lo = trunc(b.instant(False) - a.instant(True))
        hi = trunc(b.instant(True) - a.instant(False))
    if not (-(1 << 31) <= lo and hi < (1 << 31)):
        raise Leave()
    return integer(lo) if lo == hi else f"Uncertainty[Integer {lo}, Integer {hi}]"


# ----------------------------------------------------------------------------
# the values the grid crosses

DATES = [D(2020), D(2020, 3), D(2020, 3, 1)]
DTS = {
    "Y": DT(2020), "M": DT(2020, 3), "D": DT(2020, 3, 1), "H": DT(2020, 3, 1, 10),
    "N": DT(2020, 3, 1, 10, 0), "S": DT(2020, 3, 1, 10, 0, 0), "L": DT(2020, 3, 1, 10, 0, 0, 0),
    "Z": DT(2020, 3, 1, 10, 0, 0, 0, off="Z"), "P": DT(2020, 3, 1, 10, 0, 0, 0, off="+14:00"),
    "W": DT(2020, 3, 1, 10, 0, 0, 0, off="-12:00"),
}
LATER = {  # the same shapes five days on, for the high end of an interval
    "Y": DT(2020), "M": DT(2020, 3), "D": DT(2020, 3, 5), "H": DT(2020, 3, 5, 10),
    "N": DT(2020, 3, 5, 10, 0), "S": DT(2020, 3, 5, 10, 0, 0), "L": DT(2020, 3, 5, 10, 0, 0, 0),
    "Z": DT(2020, 3, 5, 10, 0, 0, 0, off="Z"), "P": DT(2020, 3, 5, 10, 0, 0, 0, off="+14:00"),
    "W": DT(2020, 3, 5, 10, 0, 0, 0, off="-12:00"),
}
LATER_DATES = [D(2020), D(2020, 3), D(2020, 3, 5)]


def day_of(v, d):
    """The same shape of DateTime on another day."""
    f = list(v.f)
    if len(f) >= 3:
        f[2] = d
    return V(v.kind, f, v.off)


rows = []  # (expression, oracle answer or None, group)
pairs = []  # (implicit definition index, explicit definition index)


class Render:
    """Writes a Date as it is, or with ToDateTime around it."""

    def __init__(self, explicit):
        self.explicit = explicit

    def v(self, x):
        if x.kind == "Date" and self.explicit:
            return "ToDateTime(" + x.literal() + ")"
        return x.literal()

    def iv(self, lo, hi):
        return "Interval[" + self.v(lo) + ", " + self.v(hi) + "]"


def safe(fn):
    try:
        return fn()
    except Leave:
        return None


# Each template: (name, builder(render, date value(s), datetime value(s)) -> expression,
# oracle(date, datetime, order) -> answer or None). order 0: the Date first, 1: the DateTime first.

def pair_in_order(order, a, b):
    return (a, b) if order == 0 else (b, a)


def sides(order, d, t):
    return pair_in_order(order, d, t)


templates = []  # (group, label, expression function, oracle function, shapes)


def add_template(group, make, oracle, dates, dts):
    templates.append((group, make, oracle, dates, dts))


ALL_DTS = list(DTS)
MID_DTS = ["D", "H", "L", "P"]
FEW_DTS = ["D", "L", "P"]
THREE_DTS = ["D", "H", "L"]

# --- comparison operators
for op, word in (("=", "="), ("<", "<"), ("~", "~"), (">=", ">=")):
    def make(r, order, d, t, op=op):
        a, b = sides(order, d, t)
        return f"{r.v(a)} {op} {r.v(b)}"

    def oracle(order, d, t, op=op):
        a, b = sides(order, d, t)
        if op == "~":
            x = eq3(a, b)
            return answer(False if x is None else x)
        return answer(rel(a, b, op))
    add_template("compare", make, oracle, DATES, ALL_DTS)

# --- between: a Date operand between DateTime bounds, a DateTime between Dates
def make_between(r, order, d, t):
    if order == 0:
        return f"{r.v(d)} between {r.v(t)} and {r.v(day_of(t, 5))}"
    return f"{r.v(t)} between {r.v(d)} and {r.v(day_of(d, 5) if d.prec == 3 else d)}"


def oracle_between(order, d, t):
    if order == 0:
        lo, hi = t, day_of(t, 5)
        return answer(and3(rel(d, lo, ">="), rel(d, hi, "<=")))
    lo, hi = d, (day_of(d, 5) if d.prec == 3 else d)
    return answer(and3(rel(t, lo, ">="), rel(t, hi, "<=")))


add_template("between", make_between, oracle_between, DATES, ALL_DTS)

# --- time phrases
PHRASES = [
    ("same day as", lambda a, b: (lambda c: None if c is None else c == 0)(cmp(a, b, 3))),
    ("before", lambda a, b: rel(a, b, "<")),
    ("after day of", lambda a, b: (lambda c: None if c is None else c > 0)(cmp(a, b, 3))),
    ("same or before", lambda a, b: rel(a, b, "<=")),
    ("within 3 days of", None),
]
for word, fn in PHRASES:
    def make(r, order, d, t, word=word):
        a, b = sides(order, d, t)
        return f"{r.v(a)} {word} {r.v(b)}"

    def oracle(order, d, t, fn=fn):
        if fn is None:
            raise Leave()
        a, b = sides(order, d, t)
        return answer(fn(a, b))
    add_template("phrase", make, oracle, DATES, MID_DTS)

# --- a point in an interval, in the direction the translator resolves
for word, cap in (("in", 7), ("during", 7), ("in day of", 3)):
    def make(r, order, d, t, word=word):
        return f"{r.v(d)} {word} {r.iv(t, day_of(t, 5))}"

    def oracle(order, d, t, cap=cap):
        return answer(and3(rel(d, t, ">=", cap), rel(d, day_of(t, 5), "<=", cap)))
    add_template("point-in-interval", make, oracle, DATES, FEW_DTS)
for word in ("includes", "contains"):
    def make(r, order, d, t, word=word):
        return f"{r.iv(t, day_of(t, 5))} {word} {r.v(d)}"

    def oracle(order, d, t):
        return answer(and3(rel(d, t, ">="), rel(d, day_of(t, 5), "<=")))
    add_template("point-in-interval", make, oracle, DATES, FEW_DTS)


def make_dt_in_dates(r, order, d, t):
    return f"{r.v(t)} in {r.iv(d, day_of(d, 5) if d.prec == 3 else d)}"


def oracle_dt_in_dates(order, d, t):
    hi = day_of(d, 5) if d.prec == 3 else d
    return answer(and3(rel(t, d, ">="), rel(t, hi, "<=")))


add_template("point-in-interval", make_dt_in_dates, oracle_dt_in_dates, DATES, FEW_DTS)

# --- a point and a list
LIST_FORMS = [
    ("{d} in {{ {t} }}", lambda r, d, t: f"{r.v(d)} in {{ {r.v(t)} }}", lambda d, t: answer(eq3(d, t) is True)),
    ("{t} in {{ {d} }}", lambda r, d, t: f"{r.v(t)} in {{ {r.v(d)} }}", lambda d, t: answer(eq3(t, d) is True)),
    ("{{ {t} }} contains {d}", lambda r, d, t: f"{{ {r.v(t)} }} contains {r.v(d)}", lambda d, t: answer(eq3(d, t) is True)),
    ("{{ {d} }} contains {t}", lambda r, d, t: f"{{ {r.v(d)} }} contains {r.v(t)}", lambda d, t: answer(eq3(t, d) is True)),
]
for _, mk, orc in LIST_FORMS:
    add_template("list", (lambda r, order, d, t, mk=mk: mk(r, d, t)), (lambda order, d, t, orc=orc: orc(d, t)), DATES, ["D", "L"])

# --- two intervals, a Date interval and a DateTime one, in both orders
IV_OPS = [
    ("overlaps", lambda a, b: and3(rel(a[0], b[1], "<="), rel(b[0], a[1], "<="))),
    ("overlaps day of", None),
    ("during", lambda a, b: and3(rel(a[0], b[0], ">="), rel(a[1], b[1], "<="))),
    ("includes", lambda a, b: and3(rel(a[0], b[0], "<="), rel(a[1], b[1], ">="))),
    ("starts", lambda a, b: and3(eq3(a[0], b[0]), rel(a[1], b[1], "<="))),
    ("ends", lambda a, b: and3(eq3(a[1], b[1]), rel(a[0], b[0], ">="))),
    ("=", lambda a, b: and3(eq3(a[0], b[0]), eq3(a[1], b[1]))),
    ("properly includes", lambda a, b: and3(and3(rel(a[0], b[0], "<="), rel(a[1], b[1], ">=")),
                                            not3(and3(eq3(a[0], b[0]), eq3(a[1], b[1]))))),
    ("meets", None),
    ("union", None),
    ("intersect", None),
    ("except", None),
]
for word, fn in IV_OPS:
    def make(r, order, d, t, word=word):
        ivd = r.iv(d, day_of(d, 5) if d.prec == 3 else d)
        ivt = r.iv(t, day_of(t, 5))
        a, b = pair_in_order(order, ivd, ivt)
        return f"{a} {word} {b}"

    def oracle(order, d, t, word=word, fn=fn):
        if word == "overlaps day of":
            a = (d, day_of(d, 5) if d.prec == 3 else d)
            b = (t, day_of(t, 5))
            if order == 1:
                a, b = b, a
            return answer(and3(rel(a[0], b[1], "<=", 3), rel(b[0], a[1], "<=", 3)))
        if fn is None:
            raise Leave()
        a = (d, day_of(d, 5) if d.prec == 3 else d)
        b = (t, day_of(t, 5))
        if order == 1:
            a, b = b, a
        return answer(fn(a, b))
    add_template("intervals", make, oracle, DATES, ["D", "L", "P"])

# --- `between` in the units of a difference and a duration
for unit in ("years", "months", "days", "hours"):
    for form in ("difference", "duration"):
        def make(r, order, d, t, unit=unit, form=form):
            a, b = sides(order, d, t)
            if form == "difference":
                return f"difference in {unit} between {r.v(a)} and {r.v(b)}"
            return f"{unit} between {r.v(a)} and {r.v(b)}"

        def oracle(order, d, t, unit=unit, form=form):
            a, b = sides(order, d, t)
            return measure(a, b, unit, form)
        add_template("measure", make, oracle, [DATES[0], DATES[2]], THREE_DTS)

# --- a list, an interval, an if, a case, Coalesce and the aggregates: the type
# of what they answer, as the DateTime they are for the translator.
TYPE_FORMS = [
    lambda r, d, t, o: f"({{ {', '.join(r.v(x) for x in pair_in_order(o, d, t))} }})[0] is DateTime",
    lambda r, d, t, o: f"({{ {', '.join(r.v(x) for x in pair_in_order(o, d, t))} }})[1] is DateTime",
    lambda r, d, t, o: f"(if true then {r.v(d)} else {r.v(t)}) is DateTime" if o == 0 else f"(if false then {r.v(t)} else {r.v(d)}) is DateTime",
    lambda r, d, t, o: f"(case when true then {r.v(d)} else {r.v(t)} end) is DateTime" if o == 0 else f"(case when true then {r.v(t)} else {r.v(d)} end) is DateTime",
    lambda r, d, t, o: f"Coalesce({r.v(d)}, {r.v(t)}) is DateTime" if o == 0 else f"Coalesce({r.v(t)}, {r.v(d)}) is DateTime",
    lambda r, d, t, o: f"First({{ {', '.join(r.v(x) for x in pair_in_order(o, d, t))} }}) is DateTime",
    lambda r, d, t, o: f"start of Interval[{r.v(d)}, {r.v(day_of(t, 5))}] is DateTime" if o == 0 else f"end of Interval[{r.v(t)}, {r.v(day_of(d, 5) if d.prec == 3 else d)}] is DateTime",
]
def type_oracle(i):
    def oracle(order, d, t):
        # an interval with a Date bound at the end may be invalid (its end before
        # its start, or not comparable), which is not what the row asks
        if i == len(TYPE_FORMS) - 1 and (order == 1 or d.prec != 3):
            raise Leave()
        return answer(True)
    return oracle


for i, mk in enumerate(TYPE_FORMS):
    add_template("type", (lambda r, order, d, t, mk=mk: mk(r, d, t, order)), type_oracle(i), DATES, ["L"])

# --- lists of the two, a union, an intersect and an except, which the translator does not convert
for word in ("union", "intersect", "except"):
    def make(r, order, d, t, word=word):
        a, b = pair_in_order(order, "{ " + r.v(d) + " }", "{ " + r.v(t) + " }")
        return f"{a} {word} {b}"
    add_template("lists", make, lambda order, d, t: (_ for _ in ()).throw(Leave()), DATES, ["D", "L"])

# ----------------------------------------------------------------------------
# the rows of the probe: the translator was measured on these (scratchpad r019,
# elmconv.tsv, elmconv-c.tsv); the oracle leaves them to the references

PROBE_D, PROBE_T = "@2020-03-01", "@2020-03-01T10:00:00.000"
PROBE_D2, PROBE_T2 = "@2020-03-05", "@2020-03-05T10:00:00.000"


def probe_rows(d, t, d2, t2):
    ivd, ivt = f"Interval[{d}, {d2}]", f"Interval[{t}, {t2}]"
    out = []
    for op in ("=", "!=", "~", "!~", "<", ">="):
        out += [f"{d} {op} {t}", f"{t} {op} {d}"]
    out += [f"{d} between {t} and {t}", f"{t} between {d} and {d}"]
    for word in ("same day as", "before", "after day of", "same or before", "within 3 days of", "1 day or less before"):
        out += [f"{d} {word} {t}", f"{t} {word} {d}"]
    out += [f"{d} in {ivt}", f"{ivt} in {d}", f"{d} during {ivt}", f"{ivt} during {d}",
            f"{ivt} includes {d}", f"{d} includes {ivt}", f"{ivt} contains {d}", f"{d} contains {ivt}",
            f"{d} in {{ {t} }}", f"{t} in {{ {d} }}", f"{{ {t} }} contains {d}", f"{{ {d} }} contains {t}",
            f"{{ {d}, {t} }}", f"{{ {t}, {d} }}", f"Interval[{d}, {t}]", f"Interval[{t}, {d}]",
            f"if true then {d} else {t}", f"if true then {t} else {d}",
            f"case when true then {d} else {t} end", f"case when true then {t} else {d} end",
            f"Coalesce({d}, {t})", f"Coalesce({t}, {d})",
            f"Coalesce(null, {d}, {t})", f"Coalesce(null, {t}, {d})", f"Coalesce({d}, null, {t})",
            f"Coalesce({{ {d} }}, {{ {t} }})"]
    for op in ("overlaps", "meets", "starts", "ends", "during", "includes", "properly includes", "=",
               "union", "intersect", "except", "overlaps day of"):
        out += [f"{ivd} {op} {ivt}", f"{ivt} {op} {ivd}"]
    for op in ("union", "intersect", "except"):
        out += [f"{{ {d} }} {op} {{ {t} }}", f"{{ {t} }} {op} {{ {d} }}"]
    out += [f"{{ {d} }} = {{ {t} }}", f"{{ {t} }} = {{ {d} }}",
            f"days between {d} and {t}", f"days between {t} and {d}",
            f"difference in days between {d} and {t}", f"difference in days between {t} and {d}",
            f"difference in hours between @2012-01-01 and @2012-01-02T05:00",
            f"hours between @2012-01-01 and @2012-01-02T05:00",
            f"days between @2012-01-01 and @2012-01-05T00:00:00.000",
            f"days between @2012-01-01 and @2012-01-05T10:00:00.000",
            f"duration in days of Interval[{d}, {t}]", f"duration in days of Interval[{t}, {d}]",
            f"Max({{ {d}, {t} }})", f"Max({{ {t}, {d} }})", f"Min({{ {d}, {t} }})", f"Count({{ {d}, {t} }})",
            f"Mode({{ {d}, {t} }})", f"distinct {{ {d}, {t} }}", f"First({{ {d}, {t} }})",
            f"{d} in day of {ivt}", f"{ivt} in day of {d}",
            f"{ivd} overlaps day of {ivt}", f"{ivt} overlaps day of {ivd}",
            f"F({d})", f"G({ivd})", f"F({t})",
            f"MP", f"P", f"{ivt} during MP",
            f"{{ {{ {d} }}, {{ {t} }} }}", f"{{ Interval[{d}, {d}], Interval[{t}, {t}] }}",
            f"{{ Tuple {{ a: {d} }}, Tuple {{ a: {t} }} }}",
            f"({{ {d} }}) X where X < {t}", f"({{ {d} }}) X return X = {t}", f"({{ {d}, {t} }}) X return X",
            f"({{ {d} }}) X let Y: {t} return X < Y", f"({{ {t} }}) X where X > {d}",
            f"{{ {d}, null, {t} }}", f"Interval[{d}, null] contains {t}", f"Interval[null, {t}] contains {d}",
            f"{{ {d}, {t} }} includes {d}", f"{d} in {{ {t}, null }}",
            f"collapse {{ Interval[{d}, {d}], Interval[{t}, {t}] }}",
            f"expand {{ Interval[{d}, {d}] }} per day = {{ Interval[{t}, {t}] }}",
            f"{d} as DateTime", f"{d} in Interval[{t}, null]", f"{d} in Interval[null, {t}]"]
    return out


# the probe rows with every Date written as ToDateTime: the same ELM for the translator
def explicit_probe(d, t, d2, t2):
    ed, ed2 = f"ToDateTime({d})", f"ToDateTime({d2})"
    rows_ = probe_rows(ed, t, ed2, t2)
    return rows_


HEADER = """library Differential version '1.0'
using FHIR version '4.0.1'
include FHIRHelpers version '4.0.1'
parameter MP Interval<DateTime> default Interval[@2020-03-01, @2020-03-05]
parameter P DateTime default @2020-03-01
define function F(x DateTime): x
define function G(x Interval<DateTime>): x"""


def build():
    out = []  # (expression, oracle answer or None, group, role)
    probe = probe_rows(PROBE_D, PROBE_T, PROBE_D2, PROBE_T2)
    explicit = explicit_probe(PROBE_D, PROBE_T, PROBE_D2, PROBE_T2)
    base = {}
    for i, expr in enumerate(probe):
        base[i] = len(out)
        out.append((expr, None, "probe", None))
    n_probe = len(out)
    grid = []
    for group, make, oracle, dates, dts in templates:
        for d in dates:
            for key in dts:
                t = DTS[key]
                for order in (0, 1):
                    expr = make(Render(False), order, d, t)
                    grid.append((expr, safe(lambda: oracle(order, d, t)), group, (make, order, d, t, key)))
    for expr, ans, group, ctx in grid:
        out.append((expr, ans, group, ctx))
    # the same rows with the ToDateTime written out: all of the probe, and every
    # sixth row of the grid
    links = []
    for i, expr in enumerate(explicit):
        links.append((base[i], len(out)))
        out.append((expr, None, "explicit", None))
    for j, (expr, ans, group, ctx) in enumerate(grid):
        if j % 6 == 0:
            make, order, d, t, key = ctx
            links.append((n_probe + j, len(out)))
            out.append((make(Render(True), order, d, t), None, "explicit", None))
    return out, links, n_probe


cases, links, n_probe = build()

if "--stats" in sys.argv:
    import collections
    total, decided = collections.Counter(), collections.Counter()
    for _, ans, group, _ in cases:
        total[group] += 1
        decided[group] += ans is not None
    for group in total:
        print(f"{group}\t{total[group]}\t{decided[group]}")
    sys.exit(0)
if "--oracle" in sys.argv:
    for i, (_, ans, _, _) in enumerate(cases):
        if ans is not None:
            print(f"D{i}\t{ans}")
    sys.exit(0)
if "--pairs" in sys.argv:
    for a, b in links:
        print(f"D{a}\tD{b}")
    sys.exit(0)
if "--cases" in sys.argv:
    # What the translator accepts: it refuses a point as the container of an
    # interval and an interval as the element (elmconv.tsv D27, D29, D31, D33,
    # D85), and `as DateTime` of a Date (elmconv-c D22), and one error stops the
    # translation of a library.
    d, t = PROBE_D, PROBE_T
    ivt = f"Interval[{t}, {PROBE_T2}]"
    refused = {f"{ivt} in {d}", f"{ivt} during {d}", f"{d} includes {ivt}", f"{d} contains {ivt}",
               f"{ivt} in day of {d}", f"{d} as DateTime"}
    print(HEADER)
    for i in range(n_probe):
        if cases[i][0] not in refused:
            print(f"define D{i}: {cases[i][0]}")
    sys.exit(0)

print(HEADER)
print("""// Generated by scripts/mixeddates-area.py: the cases of
// docs/plans/019-date-promotion.md. A Date next to a DateTime is converted to a
// DateTime (CQL 1.5.3, Developer's Guide, Implicit Conversions, table 3-J).
// First the rows of the probe, where the reference translator was measured; then
// a grid of the shapes of a Date (year, month, day), those of a DateTime (year to
// millisecond, with no offset, Z, +14:00 and -12:00), the operators and the order
// of their operands; then the same rows with the ToDateTime written out.""")
for i, (expr, _, _, _) in enumerate(cases):
    print(f"define D{i}: {expr}")
