#!/usr/bin/env python3
# Writes testdata/snapshots/expansions.cql, the area of plan 026
# (docs/plans/026-expand-truncates-to-per.md), to standard output; with --oracle
# the answers its independent oracle gives, to standard output as well.
# Usage: python3 scripts/expansions-area.py > testdata/snapshots/expansions.cql
#        python3 scripts/expansions-area.py --oracle > testdata/expansions/oracle.tsv
#
# The oracle is pure Python and shares nothing with the engine. It follows CQL
# 1.5.3, Expand (https://cql.hl7.org/N1A/09-b-cqlreference.html#expand), as the
# plan reads it, with the user's decisions D1 and D2 of 16-11-2026:
# - per omitted is one unit of the coarsest precision of the boundaries of the
#   whole list; a week is seven days;
# - each bound is first its start or end (the successor or predecessor of an
#   open one, at its own precision), then truncated to the precision of per:
#   a temporal in its own digits, keeping its offset; a number toward zero (D1);
# - the list drops its nulls, is ordered, and the truncated intervals that
#   overlap or meet at the precision of per are merged, as both references do;
# - a temporal bound less precise than per contributes nothing;
# - the pieces of size per are counted from the truncated start of the first
#   interval (D2), and an interval contributes the whole pieces inside its
#   truncated bounds;
# - the point overload is the starts of the pieces.
# A temporal is an ordinal at a precision (Julian day numbers for days), so the
# calendar is never walked by adding; a number is an integer at per's scale.
import random
import sys
from fractions import Fraction

PLAN = "docs/plans/026-expand-truncates-to-per.md"

# ---------------------------------------------------------------------------
# Temporal values

DT_UNITS = ["year", "month", "day", "hour", "minute", "second", "millisecond"]
TIME_UNITS = ["hour", "minute", "second", "millisecond"]


def dim(y, m):
    if m == 2:
        return 29 if (y % 4 == 0 and y % 100 != 0) or y % 400 == 0 else 28
    return 30 if m in (4, 6, 9, 11) else 31


def jdn(y, m, d):
    a = (14 - m) // 12
    yy = y + 4800 - a
    mm = m + 12 * a - 3
    return d + (153 * mm + 2) // 5 + 365 * yy + yy // 4 - yy // 100 + yy // 400 - 32045


def from_jdn(j):
    a = j + 32044
    b = (4 * a + 3) // 146097
    c = a - 146097 * b // 4
    d = (4 * c + 3) // 1461
    e = c - 1461 * d // 4
    m = (5 * e + 2) // 153
    day = e - (153 * m + 2) // 5 + 1
    month = m + 3 - 12 * (m // 10)
    year = 100 * b + d - 4800 + m // 10
    return year, month, day


SCALE = [None, None, None, 24, 60, 60, 1000]  # sub-day steps for DateTime


class T:
    """A Date, a DateTime or a Time: kind, its fields to its precision, and the
    offset a DateTime writes ("" for none)."""

    def __init__(self, kind, fields, off=""):
        self.kind, self.f, self.off = kind, tuple(fields), off

    @property
    def units(self):
        return TIME_UNITS if self.kind == "Time" else DT_UNITS

    @property
    def prec(self):  # index of the finest unit it has
        return len(self.f) - 1

    def ordinal(self, p):
        """The ordinal of the value at precision p (p <= its own)."""
        f = self.f
        if self.kind == "Time":
            o = f[0]
            for i, s in zip(range(1, p + 1), (60, 60, 1000)):
                o = o * s + f[i]
            return o
        if p == 0:
            return f[0]
        if p == 1:
            return f[0] * 12 + f[1] - 1
        o = jdn(f[0], f[1], f[2])
        for i in range(3, p + 1):
            o = o * SCALE[i] + f[i]
        return o

    def at(self, p, o):
        """The value of the same kind and offset at precision p and ordinal o, or
        None out of the calendar or the clock."""
        if self.kind == "Time":
            fields = []
            for s in reversed((60, 60, 1000)[:p]):
                o, r = divmod(o, s)
                fields.append(r)
            if not 0 <= o <= 23:
                return None
            return T("Time", [o] + list(reversed(fields)), self.off)
        if p == 0:
            return T(self.kind, [o], self.off) if 1 <= o <= 9999 else None
        if p == 1:
            y, m = divmod(o, 12)
            return T(self.kind, [y, m + 1], self.off) if 1 <= y <= 9999 else None
        sub = []
        for i in range(p, 2, -1):
            o, r = divmod(o, SCALE[i])
            sub.append(r)
        y, m, d = from_jdn(o)
        if not 1 <= y <= 9999:
            return None
        return T(self.kind, [y, m, d] + list(reversed(sub)), self.off)

    def cql(self):
        f = self.f
        if self.kind == "Time":
            s = "@T%02d" % f[0]
            for i, v in enumerate(f[1:], 1):
                s += (":%02d" if i < 3 else ".%03d") % v
            return s
        s = "@%04d" % f[0]
        if len(f) > 1:
            s += "-%02d" % f[1]
        if len(f) > 2:
            s += "-%02d" % f[2]
        if self.kind == "DateTime":
            s += "T"
            if len(f) > 3:
                s += "%02d" % f[3]
                for i, v in enumerate(f[4:], 4):
                    s += (":%02d" if i < 6 else ".%03d") % v
            s += self.off
        return s

    def text(self):
        """The canonical text of the answer: as the engine writes it, with a
        DateTime coarser than the hour written with neither T nor offset."""
        f = self.f
        if self.kind == "Time":
            s = "%02d" % f[0]
            for i, v in enumerate(f[1:], 1):
                s += (":%02d" if i < 3 else ".%03d") % v
            return "Time " + s
        s = "%04d" % f[0]
        if len(f) > 1:
            s += "-%02d" % f[1]
        if len(f) > 2:
            s += "-%02d" % f[2]
        if self.kind == "DateTime" and len(f) > 3:
            s += "T%02d" % f[3]
            for i, v in enumerate(f[4:], 4):
                s += (":%02d" if i < 6 else ".%03d") % v
            s += self.off
        return self.kind + " " + s


def unit_index(kind, unit):
    return (TIME_UNITS if kind == "Time" else DT_UNITS).index(unit)


# ---------------------------------------------------------------------------
# Numbers


def dec_text(fr, scale):
    """A Fraction known to be a multiple of 10^-scale, written with that scale."""
    n = fr * 10 ** scale
    assert n.denominator == 1
    n = n.numerator
    sign = "-" if n < 0 else ""
    n = abs(n)
    if scale == 0:
        return f"{sign}{n}"
    s = str(n).rjust(scale + 1, "0")
    return f"{sign}{s[:-scale]}.{s[-scale:]}"


def trunc_to(fr, scale):
    """Truncated toward zero at scale (D1), as an integer count of 10^-scale."""
    n = fr * 10 ** scale
    q = abs(n.numerator) // n.denominator
    return q if n >= 0 else -q


EPS = Fraction(1, 10 ** 8)  # the step of a Decimal (CQL 1.5.3, Decimal: scale 8)


# ---------------------------------------------------------------------------
# One expansion


class Iv:
    def __init__(self, lo, hi, lc=True, hc=True):
        self.lo, self.hi, self.lc, self.hc = lo, hi, lc, hc


def temporal_bounds(iv, p):
    """The truncated start and end of iv at precision p, as ordinals, or None if
    a bound is less precise than p."""
    lo, hi = iv.lo, iv.hi
    if lo.prec < p or hi.prec < p:
        return None
    s = lo.ordinal(lo.prec) + (0 if iv.lc else 1)
    e = hi.ordinal(hi.prec) - (0 if iv.hc else 1)
    s_val, e_val = lo.at(lo.prec, s), hi.at(hi.prec, e)
    assert s_val is not None and e_val is not None
    return s_val.ordinal(p), e_val.ordinal(p), s_val, e_val


RULE = {"low": "trunc", "high": "trunc", "anchor": "first", "end8": False}


def ceil_ordinal(v, p):
    """The ordinal at p of the first unit of p that starts at v or after it."""
    o = v.ordinal(p)
    t = v.at(p, o)
    floor = list(t.f) + [(1 if (v.kind != "Time" and i in (1, 2)) else 0) for i in range(p + 1, v.prec + 1)]
    return o if tuple(floor) == v.f else o + 1


def merge(spans, reach):
    """The truncated spans in order, those that overlap or meet at the precision
    of per merged (reach 1); cql-execution merges those within a whole per
    (reach = step)."""
    out = []
    for lo, hi in sorted(spans):
        if lo > hi:
            continue
        if out and lo <= out[-1][1] + reach:
            out[-1][1] = max(out[-1][1], hi)
        else:
            out.append([lo, hi])
    return out


def first_k(anchor, lo, step, how):
    if how == "grid":
        return (lo - anchor) // step  # the piece of the grid the low is in
    return -((anchor - lo) // step)  # the first piece that starts at the low or after


def expand_temporal(ivs, unit, amount, points, m=RULE):
    """The answer text of expand over a list of temporal intervals. m is the
    reading: RULE, the plan's; the others are the references' (--verdicts)."""
    kind = ivs[0].lo.kind
    week = unit == "week"
    p = unit_index(kind, "day" if week else unit)
    step = amount * (7 if week else 1)
    spans = []
    proto = None
    for iv in ivs:
        if iv.lo.prec < p or iv.hi.prec < p:
            continue  # contributes nothing
        # The start and the end at each bound's own precision, then truncated.
        s = iv.lo.at(iv.lo.prec, iv.lo.ordinal(iv.lo.prec) + (0 if iv.lc else 1))
        e = iv.hi.at(iv.hi.prec, iv.hi.ordinal(iv.hi.prec) - (0 if iv.hc else 1))
        proto = proto or s
        spans.append((ceil_ordinal(s, p) if m["low"] == "ceil" else s.ordinal(p), e.ordinal(p)))
    out = []
    cut = merge(spans, step if m["anchor"] == "collapse" else 1)
    if cut:
        anchor = cut[0][0]
        for lo, hi in cut:
            if m["anchor"] in ("each", "collapse"):
                anchor = lo
            k = first_k(anchor, lo, step, m["anchor"])
            while True:
                start = anchor + k * step
                end = start + step - 1
                if end > hi:
                    break
                a, b = proto.at(p, start), proto.at(p, end)
                assert a is not None and b is not None
                out.append(a.text() if points else f"Interval[{a.text()}, {b.text()}]")
                k += 1
    return "List[" + ", ".join(out) + "]"


def floor_to(fr, scale):
    n = fr * 10 ** scale
    return n.numerator // n.denominator


def expand_numbers(kind, ivs, per, points, unit="", m=RULE):
    """kind is Integer, Decimal or Quantity; bounds are Fractions; per a Fraction
    that is a whole number of units of its own scale."""
    q = 0
    while (per * 10 ** q).denominator != 1:
        q += 1
    step = int(per * 10 ** q)
    spans = []
    for iv in ivs:
        if kind == "Integer":
            s = iv.lo + (0 if iv.lc else 1)
            e = iv.hi - (0 if iv.hc else 1)
            # An Integer gains precision: its high is the last value of its unit.
            lo, hi = int(s) * 10 ** q, int(e) * 10 ** q + 10 ** q - 1
        else:
            s = iv.lo + (0 if iv.lc else EPS)
            e = iv.hi - (0 if iv.hc else EPS)
            lo = -floor_to(-s, q) if m["low"] == "ceil" else trunc_to(s, q)
            hi = floor_to(e, q) if m["high"] == "floor" else trunc_to(e, q)
        spans.append((lo, hi))
    merged = merge(spans, step if m["anchor"] == "collapse" else 1)
    out = []
    typ = "Integer" if kind == "Integer" and q == 0 else ("Quantity" if kind == "Quantity" else "Decimal")
    def word(n):
        v = dec_text(Fraction(n, 10 ** q), q)
        if typ == "Quantity":
            return f"Quantity {v} '{unit}'"
        return f"{typ} {v}"
    if merged:
        anchor = merged[0][0]
        for lo, hi in merged:
            if m["anchor"] in ("each", "collapse"):
                anchor = lo
            k = first_k(anchor, lo, step, m["anchor"])
            while True:
                start = anchor + k * step
                end = start + step - 1
                # cql-execution's piece ends 10^-8 before the next, and is whole
                # when that is not past the high bound.
                if (start + step if m["end8"] and typ != "Integer" else end) > hi:
                    break
                if m["end8"] and typ != "Integer":
                    # A piece that ends 10^-8 before the next starts.
                    last = dec_text(Fraction(start + step, 10 ** q) - EPS, 8)
                    out.append(word(start) if points else f"Interval[{word(start)}, {last}]")
                else:
                    out.append(word(start) if points else f"Interval[{word(start)}, {word(end)}]")
                k += 1
    return "List[" + ", ".join(out) + "]"


# ---------------------------------------------------------------------------
# The rows


def iv_cql(iv, num=None):
    lo = iv.lo.cql() if num is None else num(iv.lo)
    hi = iv.hi.cql() if num is None else num(iv.hi)
    return f"Interval{'[' if iv.lc else '('}{lo}, {hi}{']' if iv.hc else ')'}"


def per_cql(unit, amount):
    if unit is None:
        return ""
    if unit == "week":
        return " per week" if amount == 1 else f" per {amount} weeks"
    return f" per {unit}" if amount == 1 else f" per {amount} {unit}s"


rows = []  # (expression, oracle text or None)


def add(expr, ans, model=None):
    # A long expansion says nothing a short one does not, and costs the three
    # engines and the files: the area keeps answers of a few dozen pieces.
    if ans is not None and len(ans) > 4000:
        return
    rows.append((expr, ans))
    models.append(model)


models = []  # per row, the answer under a reading: model(m, points) -> text


def coarsest(ivs):
    return min(min(iv.lo.prec, iv.hi.prec) for iv in ivs)


def temporal_row(ivs, unit, amount, points=False):
    kind = ivs[0].lo.kind
    if points:
        expr = f"expand {iv_cql(ivs[0])}{per_cql(unit, amount)}"
    else:
        expr = "expand { " + ", ".join(iv_cql(iv) for iv in ivs) + " }" + per_cql(unit, amount)
    u = unit if unit is not None else (TIME_UNITS if kind == "Time" else DT_UNITS)[coarsest(ivs)]
    step = amount if unit is not None else 1
    if kind == "Time":
        # The Java engine 5.4.0 walks a Time past the end of the day, where it
        # wraps to the morning and never passes the high bound, and runs out of
        # memory (D535 and D678 of an earlier draft of this area). A row whose walk
        # would cross midnight leaves the area; engine_expand_per_test.go holds
        # those cases (TestTheCalendarIsWalkedFromTheStart).
        p = unit_index(kind, u)
        day = (24, 24 * 60, 24 * 3600, 24 * 3600 * 1000)[p]
        highs = [iv.hi.ordinal(p) for iv in ivs if iv.hi.prec >= p]
        if highs and max(highs) + 2 * step >= day:
            return
    add(expr, expand_temporal(ivs, u, step, points),
        lambda m, pts=points: expand_temporal(ivs, u, step, pts, m))


def number_row(kind, ivs, per, per_text, points=False, unit=""):
    def num(v):
        if kind == "Integer":
            return str(v)
        scale = 1  # a Decimal literal needs its point, or it is an Integer
        while (v * 10 ** scale).denominator != 1:
            scale += 1
        return dec_text(v, scale) + (f" '{unit}'" if unit else "")
    if points:
        expr = f"expand {iv_cql(ivs[0], num)}{per_text}"
    else:
        expr = "expand { " + ", ".join(iv_cql(iv, num) for iv in ivs) + " }" + per_text
    add(expr, expand_numbers(kind, ivs, per, points, unit),
        lambda m, pts=points: expand_numbers(kind, ivs, per, pts, unit, m))


rnd = random.Random(26)


def rand_dt(kind, prec, off, base_year=2020):
    if kind == "Time":
        f = [rnd.randint(0, 22), rnd.randint(0, 59), rnd.randint(0, 59), rnd.randint(0, 999)]
        return T("Time", f[: prec + 1])
    y = base_year
    m = rnd.randint(1, 12)
    d = rnd.randint(1, dim(y, m))
    f = [y, m, d, rnd.randint(0, 23), rnd.randint(0, 59), rnd.randint(0, 59), rnd.randint(0, 999)]
    return T(kind, f[: prec + 1], off if kind == "DateTime" and prec >= 3 else "")


def later(v, p, units):
    """v moved later by `units` of precision p (p <= its precision), keeping its
    finer fields where the calendar allows."""
    o = v.ordinal(p) + units
    w = v.at(p, o)
    if w is None:
        return None
    fields = list(w.f) + list(v.f[p + 1:])
    if v.kind != "Time" and len(fields) > 2:
        fields[2] = min(fields[2], dim(fields[0], fields[1]))
    return T(v.kind, fields, v.off)


def le(a, b):
    return a.f <= b.f  # one kind, precision and offset: the fields order them


def build():
    # 1. The examples of the specification.
    add("expand { Interval[@2018-01-01, @2018-01-04] } per day",
        expand_temporal([Iv(T("Date", [2018, 1, 1]), T("Date", [2018, 1, 4]))], "day", 1, False))
    add("expand { Interval[@2018-01-01, @2018-01-04] } per 2 days",
        expand_temporal([Iv(T("Date", [2018, 1, 1]), T("Date", [2018, 1, 4]))], "day", 2, False))
    add("expand { Interval[@T10:00, @T12:30] } per hour",
        expand_temporal([Iv(T("Time", [10, 0]), T("Time", [12, 30]))], "hour", 1, False))
    add("expand { Interval[@T10, @T10] } per minute",
        expand_temporal([Iv(T("Time", [10]), T("Time", [10]))], "minute", 1, False))
    number_row("Decimal", [Iv(Fraction(10), Fraction(25, 2))], Fraction(1), " per 1")
    number_row("Integer", [Iv(10, 10)], Fraction(1, 10), " per 0.1")
    number_row("Integer", [Iv(1, 10)], Fraction(1), "", points=True)
    number_row("Integer", [Iv(1, 10)], Fraction(2), " per 2", points=True)

    # 2. A grid of single temporal intervals: every kind, offsets, the precision
    #    of the bounds against per, closures, both overloads.
    kinds = [("Date", ""), ("DateTime", "Z"), ("DateTime", "+05:30"), ("DateTime", ""), ("Time", "")]
    for kind, off in kinds:
        units = TIME_UNITS if kind == "Time" else DT_UNITS
        top = 3 if kind == "Time" else (2 if kind == "Date" else 6)
        for bprec in range(0 if kind != "Date" else 0, top + 1):
            if kind == "DateTime" and bprec < 2 and off:
                continue  # a DateTime coarser than the day writes no offset here
            for pidx in range(0, bprec + 1):
                unit = units[pidx]
                for amount in (1, 2, 3):
                    for _ in range(2):
                        lo = rand_dt(kind, bprec, off)
                        span = rnd.choice([0, 1, 2, 3, 5, 7])
                        hi = later(lo, pidx, span * amount + rnd.randint(0, amount))
                        if hi is None or (kind == "Time" and hi.f < lo.f):
                            continue
                        if kind != "Time" and hi.f[0] > 9999:
                            continue
                        lc, hc = rnd.choice([(True, True), (True, True), (False, True), (True, False), (False, False)])
                        if hi.f == lo.f and not (lc and hc):
                            continue
                        iv = Iv(lo, hi, lc, hc)
                        # The successor or predecessor of an open bound must exist,
                        # and the open interval must not be empty.
                        if not lc and lo.at(lo.prec, lo.ordinal(lo.prec) + 1) is None:
                            continue
                        if not hc and hi.at(hi.prec, hi.ordinal(hi.prec) - 1) is None:
                            continue
                        if (not lc or not hc) and hi.ordinal(hi.prec) - lo.ordinal(lo.prec) < 2:
                            continue
                        temporal_row([iv], unit, amount, points=rnd.random() < 0.3)
            # per finer than the bounds: nothing.
            if bprec < top:
                lo = rand_dt(kind, bprec, off)
                hi = later(lo, bprec, 2)
                if hi is not None:
                    temporal_row([Iv(lo, hi)], units[bprec + 1], 1)
    # weeks
    for _ in range(12):
        lo = rand_dt("Date", 2, "")
        hi = later(lo, 2, rnd.randint(0, 30))
        if hi is not None:
            temporal_row([Iv(lo, hi)], "week", rnd.choice([1, 1, 2]))
    # per omitted: one unit of the coarsest precision of the list.
    for kind, off in kinds:
        top = 3 if kind == "Time" else (2 if kind == "Date" else 5)
        for bprec in range(0, top + 1):
            if kind == "DateTime" and bprec < 3 and off:
                continue
            lo = rand_dt(kind, bprec, off)
            hi = later(lo, bprec, rnd.randint(0, 4))
            if hi is not None:
                temporal_row([Iv(lo, hi)], None, 1)

    # 3. Lists: several intervals of one kind and precision, shuffled, overlapping,
    #    meeting or apart, with per of one or several units, or omitted.
    for kind, off in kinds:
        top = 3 if kind == "Time" else (2 if kind == "Date" else 5)
        for bprec in range(1 if kind != "Time" else 0, top + 1):
            if kind == "DateTime" and bprec < 3 and off:
                continue
            units = TIME_UNITS if kind == "Time" else DT_UNITS
            for pidx in sorted({bprec, max(0, bprec - 1)}):
                for amount in (1, 2, 3, None):
                    base = rand_dt(kind, bprec, off)
                    ivs = []
                    cur = base
                    for _ in range(rnd.randint(2, 3)):
                        lo = later(cur, pidx, rnd.randint(-1, 3))
                        if lo is None:
                            break
                        hi = later(lo, pidx, rnd.randint(0, 4))
                        if hi is None:
                            break
                        ivs.append(Iv(lo, hi))
                        cur = hi
                    if len(ivs) < 2 or any(not le(iv.lo, iv.hi) for iv in ivs):
                        continue
                    rnd.shuffle(ivs)
                    if amount is None:
                        temporal_row(ivs, None, 1)
                    else:
                        temporal_row(ivs, units[pidx], amount)

    # 4. Numbers: Integer, Decimal and Quantity of one unit, both signs, closures,
    #    per of one or several units of its scale.
    for _ in range(70):
        kind = rnd.choice(["Decimal", "Decimal", "Quantity", "Integer"])
        if kind == "Integer":
            lo = rnd.randint(-6, 6)
            hi = lo + rnd.randint(0, 8)
            per, ptext = rnd.choice([(Fraction(1), " per 1"), (Fraction(2), " per 2"), (Fraction(3), " per 3"),
                                     (Fraction(1, 10), " per 0.1")])
            lc, hc = rnd.choice([(True, True), (False, True), (True, False), (False, False)])
            if (not lc or not hc) and hi - lo < 2:
                lc = hc = True
            number_row("Integer", [Iv(lo, hi, lc, hc)], per, ptext, points=rnd.random() < 0.3)
            continue
        scale = rnd.choice([1, 2])
        lo = Fraction(rnd.randint(-600, 600), 10 ** scale)
        hi = lo + Fraction(rnd.randint(0, 600), 10 ** scale)
        options = [(Fraction(1), "1"), (Fraction(2), "2"), (Fraction(3), "3"), (Fraction(1, 10), "0.1")]
        if scale == 2:
            options.append((Fraction(1, 100), "0.01"))
        per, p = rnd.choice(options)
        if per < 1 and hi - lo > 3:
            hi = lo + Fraction(rnd.randint(0, 30), 10 ** scale)
        lc, hc = rnd.choice([(True, True), (False, True), (True, False), (False, False)])
        if lo == hi:
            lc = hc = True
        unit = "mg" if kind == "Quantity" else ""
        ptext = f" per {p}" + (f" '{unit}'" if unit else "")
        number_row(kind, [Iv(lo, hi, lc, hc)], per, ptext, points=rnd.random() < 0.3, unit=unit)
    # lists of numbers (D2 and merging)
    for _ in range(25):
        kind = rnd.choice(["Integer", "Decimal"])
        ivs = []
        cur = rnd.randint(-5, 5)
        for _ in range(rnd.randint(2, 3)):
            lo = cur + rnd.randint(-1, 3)
            hi = lo + rnd.randint(0, 5)
            if kind == "Decimal":
                ivs.append(Iv(Fraction(lo * 10 + rnd.randint(0, 9), 10), Fraction(hi * 10 + rnd.randint(0, 9), 10)))
            else:
                ivs.append(Iv(lo, hi))
            cur = hi
        if any(iv.lo > iv.hi for iv in ivs):
            continue
        rnd.shuffle(ivs)
        per, ptext = rnd.choice([(Fraction(1), " per 1"), (Fraction(2), " per 2"), (Fraction(3), " per 3")])
        number_row(kind, ivs, per, ptext)

    # 5. The edges of the calendar and the clock, written by hand.
    D = lambda *f: T("Date", f)
    Tm = lambda *f: T("Time", f)
    DT = lambda off, *f: T("DateTime", f, off)
    for ivs, unit, amount, points in [
        ([Iv(D(2020, 1, 31), D(2020, 5, 31))], "month", 1, False),
        ([Iv(D(2020, 1, 30), D(2020, 3, 2))], "month", 1, False),
        ([Iv(D(2020, 1, 31), D(2020, 3, 31))], "day", 1, True),
        ([Iv(D(2019, 2, 27), D(2019, 3, 2))], "day", 1, False),
        ([Iv(D(2020, 2, 27), D(2020, 3, 2))], "day", 1, False),
        ([Iv(D(2000, 2, 28), D(2000, 3, 1))], "day", 1, False),
        ([Iv(D(1900, 2, 28), D(1900, 3, 1))], "day", 1, False),
        ([Iv(D(9999, 12, 31), D(9999, 12, 31))], "day", 1, False),
        ([Iv(D(9999, 12, 31), D(9999, 12, 31))], "day", 1, True),
        ([Iv(D(9999, 12, 30), D(9999, 12, 31))], "day", 2, False),
        ([Iv(D(9999, 11, 15), D(9999, 12, 31))], "month", 1, False),
        ([Iv(D(9998, 6, 1), D(9999, 12, 31))], "year", 1, True),
        ([Iv(D(1, 1, 1), D(1, 1, 3))], "day", 1, False),
        ([Iv(Tm(23), Tm(23))], "hour", 1, False),
        ([Iv(Tm(22, 30), Tm(23, 59))], "hour", 1, False),
        ([Iv(Tm(23, 58), Tm(23, 59, 59, 999))], "minute", 1, False),
        ([Iv(Tm(0, 0), Tm(0, 2))], "minute", 1, True),
        ([Iv(Tm(10, 59), Tm(12, 0), False, True)], "hour", 1, False),
        ([Iv(Tm(10, 30), Tm(12, 0), True, False)], "hour", 1, False),
        ([Iv(Tm(10), Tm(12), False, True)], "hour", 1, False),
        ([Iv(DT("Z", 2020, 12, 31, 23, 30), DT("Z", 2021, 1, 1, 1, 10))], "hour", 1, False),
        ([Iv(DT("+05:30", 2020, 1, 1, 23, 30), DT("+05:30", 2020, 1, 2, 1, 10))], "hour", 1, True),
        ([Iv(DT("+05:30", 2020, 1, 1, 23, 30), DT("+05:30", 2020, 1, 2, 1, 10))], "day", 1, False),
        ([Iv(DT("", 2020, 2, 28, 10, 0), DT("", 2020, 3, 1, 5, 0))], "day", 1, False),
        ([Iv(DT("Z", 9999, 12, 31, 22, 30), DT("Z", 9999, 12, 31, 23, 59))], "hour", 1, False),
        ([Iv(DT("Z", 2020, 1, 1, 0, 59), DT("Z", 2020, 1, 1, 2, 0), False, True)], "hour", 1, False),
        ([Iv(D(2020, 1, 1), D(2020, 1, 1))], "week", 1, False),
        ([Iv(D(2020, 1, 1), D(2020, 1, 21))], "week", 1, False),
        ([Iv(D(2020, 1, 1), D(2020, 1, 21))], "week", 1, True),
        ([Iv(D(2020, 1, 1), D(2020, 2, 11))], "week", 2, False),
        ([Iv(D(2020, 1, 1), D(2020, 1, 4)), Iv(D(2020, 1, 6), D(2020, 1, 9))], "day", 2, False),
        ([Iv(D(2020, 1, 6), D(2020, 1, 9)), Iv(D(2020, 1, 1), D(2020, 1, 4))], "day", 2, False),
        ([Iv(D(2020, 1, 1), D(2020, 1, 4)), Iv(D(2020, 1, 2), D(2020, 1, 5))], "day", 2, False),
        ([Iv(T("Date", [2020, 1]), T("Date", [2020, 3]))], "day", 1, False),
        ([Iv(T("Date", [2020]), T("Date", [2021]))], "month", 1, False),
    ]:
        temporal_row(ivs, unit, amount, points)
    for ivs, per, ptext in [
        ([Iv(Fraction(-29, 10), Fraction(-11, 10))], Fraction(1), " per 1"),
        ([Iv(Fraction(-15, 10), Fraction(12, 10))], Fraction(1), " per 1"),
        ([Iv(Fraction(-20, 10), Fraction(-10, 10), False, False)], Fraction(1), " per 1"),
        ([Iv(Fraction(15, 10), Fraction(30, 10), False, False)], Fraction(1), " per 1"),
        ([Iv(Fraction(10, 10), Fraction(30, 10), False, False)], Fraction(1), " per 1"),
        ([Iv(Fraction(10, 10), Fraction(30, 10))], Fraction(2), " per 2"),
        ([Iv(Fraction(15, 10), Fraction(42, 10))], Fraction(2), " per 2"),
        ([Iv(Fraction(10), Fraction(10))], Fraction(1, 10), " per 0.1"),
    ]:
        number_row("Decimal", ivs, per, ptext)
        number_row("Decimal", ivs, per, ptext, points=True)
    for ivs, per, ptext in [
        ([Iv(1, 4), Iv(6, 9)], Fraction(2), " per 2"),
        ([Iv(6, 9), Iv(1, 4)], Fraction(2), " per 2"),
        ([Iv(1, 4), Iv(2, 5)], Fraction(2), " per 2"),
        ([Iv(2, 9), Iv(1, 1)], Fraction(3), " per 3"),
        ([Iv(1, 2), Iv(3, 3)], Fraction(1), " per 1"),
        ([Iv(1, 4)], Fraction(3), " per 3"),
    ]:
        number_row("Integer", ivs, per, ptext)
    # Not answered by the oracle: an interval with a null, and an empty or null list.
    add("expand { Interval[1, 2], null, Interval[3, 3] } per 1", None)
    add("expand { Interval[1, null) } per 1", None)
    add("expand List<Interval<Integer>>{} per 1", None)
    add("expand (null as List<Interval<Integer>>) per 1", None)


build()

# ---------------------------------------------------------------------------
# The verdicts of the divergences with the references

SPEC = "CQL 1.5.3, Expand (https://cql.hl7.org/N1A/09-b-cqlreference.html#expand)"
TRUNCATED = (f'{SPEC}: "If the interval boundaries are more precise than the per quantity, the more precise values '
             f'will be truncated to the precision specified by the per quantity"')
WHOLE = ('"contribute all the intervals of size per that start on or after the lower boundary and end on or before '
         'the upper boundary"')
ANCHORED = '"the \'per\' partitions start from the starting boundary of the first input interval, ordered"'
POINTS = ('"If the input argument is an interval, rather than a list of intervals, the result is a list of points, '
          'rather than a list of intervals"')
TYPE_WORDS = {"List", "Decimal", "Integer", "Number", "DateTime", "Date", "Time", "Quantity", "Long"}
VALUE = __import__("re").compile(r"Interval\[|[^\s,\[\]{}()]+")
DT_TOKEN = __import__("re").compile(r"^(\d{4}(?:-\d\d){0,2})(T[\d:.]*)?(Z|[+-]\d\d:\d\d)?$")


def tokens(s, offsets=True, tolerance=False):
    """The values of an answer in order, with Interval[ where one opens: type
    words and punctuation dropped, numbers by value, +00:00 as Z, and with
    offsets=False no offset at all."""
    out = []
    for tok in VALUE.findall(s):
        if tok in TYPE_WORDS:
            continue
        m = DT_TOKEN.match(tok)
        if m and (m.group(2) or "-" in m.group(1)):
            date, time, off = m.group(1), (m.group(2) or "").rstrip("T") if m.group(2) != "T" else "", m.group(3) or ""
            off = "Z" if off == "+00:00" else off
            out.append(date + time + (off if offsets and time else ""))
            continue
        try:
            v = Fraction(tok)
            if tolerance:
                v = round(v * 10 ** 7) / Fraction(10 ** 7)
            out.append(str(v))
        except (ValueError, ZeroDivisionError):
            out.append(tok)
    return out


JAVA = {"low": "ceil", "high": "floor", "anchor": "each", "end8": False}
JS = {"low": "trunc", "high": "trunc", "anchor": "collapse", "end8": False}
JS8 = {"low": "trunc", "high": "trunc", "anchor": "collapse", "end8": True}


# The rows the oracle does not answer, read by hand: an open null bound
# contributes nothing (Expand: "if the boundary is open (e.g., Interval[0, null)),
# the interval will not contribute any results to the output").
BY_HAND = {
    "expand { Interval[5.79, 5.83] } per 0.1": (
        "reference defect", "reference defect: {who} answers {refans}: it does not truncate 5.79 to 5.7 and 5.83 to "
        "5.8, the scale of per, where " + TRUNCATED + ", and a piece of 0.1 from 5.79 ends past 5.83"),
    "expand List<Interval<Integer>>{} per 1": (
        "reference defect", "reference defect: {who} answers {refans} for an empty list, where " + SPEC +
        ': "If the list of intervals is empty, the result is empty"'),
    "expand { Interval[1, null) } per 1": (
        "reference defect", "reference defect: {who} answers {refans} for a list whose only interval has an open null "
        "bound, where " + SPEC + ': "if the boundary is open (e.g., Interval[0, null)), the interval will not '
        'contribute any results to the output", so the list is empty'),
}


def verdict(name, ref, refans, eng):
    i = int(name[1:]) - 1
    expr, oracle = rows[i]
    model = models[i]
    who = "the Java engine" if ref == "java" else "cql-execution"
    if ref == "java" and expr == "expand { Interval[8.0, 13.5], Interval[3.0, 7.1] } per 2":
        return ("reference defect", "reference defect: the Java engine merges the intervals of a list before it "
                "truncates them here, so [3.0, 7.1] and [8.0, 13.5], which truncate to [3, 7] and [8, 13] and meet "
                "at the precision of per, stay apart, and it counts the second from its own start, where "
                f"{TRUNCATED} and {ANCHORED}; elsewhere it merges the truncated intervals (D652)")
    if expr in BY_HAND:
        cls, ev = BY_HAND[expr]
        return cls, ev.format(who=who, refans=refans)
    if oracle is None or model is None or tokens(oracle) != tokens(eng):
        return None  # unanswered, or the engine moved: read again once the snapshot follows
    points = not expr.startswith("expand {")
    if refans.startswith("error"):
        return ("reference defect", f"reference defect: {who} fails ({refans[:80].strip()}) where {SPEC} gives "
                f"the whole pieces of size per inside the bounds, {WHOLE}, and the last one ends at the end of the "
                f"calendar, the clock or the high bound")
    if tokens(refans, offsets=False) == tokens(eng, offsets=False):
        return ("deliberate", f"deliberate: the same values, printed otherwise: {who} writes the offset of a DateTime "
                f"another way (+00:00 for Z, the evaluation's offset for none, or none at all), and this engine keeps "
                f"a DateTime's own offset as written; decided in {PLAN}, which names this row")
    open_number = __import__("re").search(r"Interval\((-?\d)|, -?\d[^\]]*\)( |$)", expr) and "@" not in expr
    integer_fraction = __import__("re").match(r"expand \{? ?Interval[\[(]-?\d+, -?\d+[\])] \}? ?per 0\.", expr)
    if ref == "java":
        if integer_fraction:
            return ("reference defect", f"reference defect: the Java engine does not give an Integer interval the "
                    f"precision of a per of a tenth, where {SPEC}: \"For numeric intervals, adding the per to the lower "
                    f"boundary produces a more precise value for the output intervals\", with the example expand "
                    f"{{ Interval[10, 10] }} per 0.1 = {{ Interval[10.0, 10.0], Interval[10.1, 10.1], ..., "
                    f"Interval[10.9, 10.9] }}")
        if tokens(refans, offsets=False) == tokens(model(JAVA), offsets=False):
            return ("reference defect", f"reference defect: the Java engine rounds the low bound up to the next unit "
                    f"of per and the high one down, where {TRUNCATED}, and counts the pieces of each merged interval "
                    f"from its own start, where {ANCHORED}; read that way it answers what it answers here")
        return None
    as_list = points and refans.startswith("{Interval[")
    for m in (JS, JS8):
        for tol in (False, True):
            got = model(m, False if as_list else points)
            if tokens(refans, offsets=False, tolerance=tol) != tokens(got, offsets=False, tolerance=tol):
                continue
            parts = []
            if as_list:
                parts.append(f"cql-execution answers intervals for the interval overload, where {POINTS}")
            if m is JS8:
                parts.append("cql-execution ends a piece of a Decimal 10^-8 before the next, in floating point, where "
                             "the specification's example, expand { Interval[10, 10] } per 0.1, gives "
                             "Interval[10.0, 10.0], Interval[10.1, 10.1], ...")
            plain = model({**m, "anchor": "first", "end8": False}, False if as_list else points)
            if tokens(model({**m, "end8": False}, False if as_list else points), offsets=False) != tokens(plain, offsets=False):
                parts.append(f"cql-execution merges intervals of the list that are less than a whole per apart and "
                             f"counts each merged interval from its own start, so its pieces cover values that are in "
                             f"no interval of the input, where {WHOLE} and {ANCHORED}")
            if not parts:
                return None
            return ("reference defect", "reference defect: " + "; ".join(parts))
    if __import__("re").search(r"\d\.\d*(9999999|0000001)\b", refans):
        parts = []
        if as_list:
            parts.append(f"cql-execution answers intervals for the interval overload, where {POINTS}")
        parts.append("cql-execution computes the pieces of a Decimal in floating point and ends each 10^-8 before the "
                     "next, where the specification's example, expand { Interval[10, 10] } per 0.1, gives "
                     "Interval[10.0, 10.0], Interval[10.1, 10.1], ..., Interval[10.9, 10.9]; a Decimal bound is the "
                     f"point it writes, and a piece ends on or before it ({WHOLE})")
        return ("reference defect", "reference defect: " + "; ".join(parts))
    if open_number:
        return ("deliberate", f"deliberate: an open bound is its successor or predecessor at its own precision, "
                f"10^-8 for a Decimal (Start and End), and is truncated after it, toward zero (D1); {who} drops the "
                f"unit the open bound is in; decided in {PLAN} (rule 3 and D1), which names this row"
                + (f"; besides, cql-execution answers intervals for the interval overload, where {POINTS}"
                   if as_list else ""))
    return None


def verdicts(path):
    out, counts, unexplained = [], {}, []
    for line in open(path, encoding="utf-8").read().split("\n"):
        if not line or line.startswith("#"):
            out.append(line)
            continue
        f = line.split("\t")
        name, ref, refans, eng = f[0], f[1], f[5], f[6]
        v = verdict(name, ref, refans, eng)
        if v is None:
            unexplained.append(name + " " + ref)
            out.append(line)
            continue
        cls, ev = v
        counts[(ref, cls)] = counts.get((ref, cls), 0) + 1
        out.append("\t".join([name, ref, cls, eng, ev, refans, eng]))
    open(path, "w", encoding="utf-8").write("\n".join(out))
    for k in sorted(counts):
        print(k, counts[k], file=sys.stderr)
    print("unexplained:", len(unexplained), " ".join(unexplained), file=sys.stderr)


if "--verdicts" in sys.argv:
    verdicts(sys.argv[sys.argv.index("--verdicts") + 1])
    sys.exit(0)

if "--oracle" in sys.argv:
    for i, (_, ans) in enumerate(rows, 1):
        if ans is not None:
            print(f"D{i}\t{ans}")
    sys.exit(0)

print("library Differential version '1.0'")
print(f"""// Generated by scripts/expansions-area.py: the cases of {PLAN},
// where expand … per truncates the bounds to the precision of per, contributes
// the whole intervals of size per, and counts them from the first interval.""")
for i, (expr, _) in enumerate(rows, 1):
    print(f"define D{i}: {expr}")
