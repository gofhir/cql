// Command drift compares two sets of reference answers, definition by
// definition, and writes how the reference's answers moved: the report a change
// of the reference's version leaves as evidence (plan 007).
//
// Each answer is one Evaluate.java or evaluate.js writes: a typed value, null,
// or an error with its class and message. A change is one of:
//
//   - error → value, value → error, a value changed;
//   - refused → accepted, accepted → refused, where the error is the
//     translator's;
//   - error class changed, or only the message, with the same class;
//   - added or removed, a definition one set answers and the other does not.
//
// With the divergence manifests of both sets, it also counts, by class, the
// rows that are new, gone or reclassified, each with its verdict before, and
// lists each of those rows with its verdict before, now, and the answer its
// reference gives now.
//
// Usage:
//
//	drift -old <dir> -new <dir> [-ref java] [-divergences] area...
package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"maps"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
)

// The kinds of change between two answers to one definition.
const (
	same           = "same"
	messageOnly    = "message only"
	errorClass     = "error class changed"
	errorToValue   = "error → value"
	valueToError   = "value → error"
	valueChanged   = "value changed"
	refusedNow     = "accepted → refused"
	acceptedNow    = "refused → accepted"
	added          = "added"
	removed        = "removed"
	translationErr = "Translation"
)

// kinds is the order the report writes them in.
var kinds = []string{same, messageOnly, errorClass, errorToValue, valueToError, valueChanged, acceptedNow, refusedNow, added, removed}

type answer struct {
	present bool
	isError bool
	class   string
	message string
	value   any
}

func read(raw json.RawMessage) (answer, error) {
	var v any
	if err := json.Unmarshal(raw, &v); err != nil {
		return answer{}, err
	}
	if m, ok := v.(map[string]any); ok {
		if class, isError := m["error"].(string); isError {
			msg, _ := m["message"].(string)
			return answer{present: true, isError: true, class: class, message: msg}, nil
		}
	}
	return answer{present: true, value: v}, nil
}

// change is the kind of change from one answer to another.
func change(old, cur answer) string {
	switch {
	case !old.present && !cur.present:
		return same
	case !old.present:
		return added
	case !cur.present:
		return removed
	case old.isError && cur.isError:
		return errorChange(old, cur)
	case old.isError:
		if old.class == translationErr {
			return acceptedNow
		}
		return errorToValue
	case cur.isError:
		if cur.class == translationErr {
			return refusedNow
		}
		return valueToError
	case !reflect.DeepEqual(old.value, cur.value):
		return valueChanged
	}
	return same
}

// errorChange is the kind of change from one error to another.
func errorChange(old, cur answer) string {
	switch {
	case old.class != cur.class && old.class == translationErr:
		return acceptedNow // the translator accepts it now, and the run errs
	case old.class != cur.class && cur.class == translationErr:
		return refusedNow
	case old.class != cur.class:
		return errorClass
	case old.message != cur.message:
		return messageOnly
	}
	return same
}

// compare counts the change of every definition either set answers.
func compare(old, cur map[string]json.RawMessage) (map[string]int, error) {
	counts := map[string]int{}
	names := map[string]bool{}
	for n := range old {
		names[n] = true
	}
	for n := range cur {
		names[n] = true
	}
	for n := range names {
		var a, b answer
		var err error
		if raw, ok := old[n]; ok {
			if a, err = read(raw); err != nil {
				return nil, fmt.Errorf("%s: %w", n, err)
			}
		}
		if raw, ok := cur[n]; ok {
			if b, err = read(raw); err != nil {
				return nil, fmt.Errorf("%s: %w", n, err)
			}
		}
		counts[change(a, b)]++
	}
	return counts, nil
}

// verdicts are a manifest's rows: the key (name and reference) to its class.
func verdicts(r io.Reader) (map[string]string, error) {
	out := map[string]string{}
	s := bufio.NewScanner(r)
	s.Buffer(make([]byte, 0, 1<<16), 1<<22)
	for s.Scan() {
		if strings.HasPrefix(s.Text(), "#") || s.Text() == "" {
			continue
		}
		f := strings.SplitN(s.Text(), "\t", 4)
		if len(f) < 3 {
			return nil, fmt.Errorf("a row with %d fields: %q", len(f), s.Text())
		}
		out[f[0]+"\t"+f[1]] = f[2]
	}
	return out, s.Err()
}

// verdictChanges counts, by the class a row had before, the rows that are
// gone, reclassified or kept, and the new ones by the class they have now.
func verdictChanges(old, cur map[string]string) map[string]int {
	out := map[string]int{}
	for k, class := range old {
		now, ok := cur[k]
		switch {
		case !ok:
			out["gone: "+class]++
		case now != class:
			out["reclassified: "+class+" → "+now]++
		default:
			out["kept: "+class]++
		}
	}
	for k, class := range cur {
		if _, ok := old[k]; !ok {
			out["new: "+class]++
		}
	}
	return out
}

// rowChange is a manifest row that is gone, reclassified or new: its verdict
// before and now, empty where the manifest does not hold it.
type rowChange struct {
	name, ref, before, now string
}

// rowChanges lists, by key, the rows whose verdict is not the one they had.
func rowChanges(old, cur map[string]string) []rowChange {
	keys := maps.Clone(old)
	maps.Copy(keys, cur)
	var out []rowChange
	for _, k := range slices.Sorted(maps.Keys(keys)) {
		if old[k] == cur[k] {
			continue
		}
		name, ref, _ := strings.Cut(k, "\t")
		out = append(out, rowChange{name: name, ref: ref, before: old[k], now: cur[k]})
	}
	return out
}

func results(path string) (map[string]json.RawMessage, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var f struct {
		Results map[string]json.RawMessage `json:"results"`
	}
	if err := json.Unmarshal(raw, &f); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	return f.Results, nil
}

func manifest(path string) (map[string]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return verdicts(f)
}

func main() {
	oldDir := flag.String("old", "", "the directory of the answers before")
	newDir := flag.String("new", "", "the directory of the answers after")
	ref := flag.String("ref", "java", "the reference: java or js")
	withVerdicts := flag.Bool("divergences", false, "also compare the areas' divergence manifests")
	flag.Parse()
	if *oldDir == "" || *newDir == "" || flag.NArg() == 0 {
		fmt.Fprintln(os.Stderr, "usage: drift -old <dir> -new <dir> [-ref java] [-divergences] area...")
		os.Exit(2)
	}
	if err := run(os.Stdout, *oldDir, *newDir, *ref, *withVerdicts, flag.Args()); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(w io.Writer, oldDir, newDir, ref string, withVerdicts bool, areas []string) error {
	fmt.Fprintf(w, "| Area | %s |\n|---|%s\n", strings.Join(kinds, " | "), strings.Repeat("---|", len(kinds)))
	total := map[string]int{}
	for _, area := range areas {
		file := area + "." + ref + ".json"
		old, err := results(filepath.Join(oldDir, file))
		if err != nil {
			return err
		}
		cur, err := results(filepath.Join(newDir, file))
		if err != nil {
			return err
		}
		counts, err := compare(old, cur)
		if err != nil {
			return fmt.Errorf("%s: %w", file, err)
		}
		cells := make([]string, len(kinds))
		for i, k := range kinds {
			cells[i] = fmt.Sprint(counts[k])
			total[k] += counts[k]
		}
		fmt.Fprintf(w, "| %s | %s |\n", area, strings.Join(cells, " | "))
	}
	cells := make([]string, len(kinds))
	for i, k := range kinds {
		cells[i] = fmt.Sprint(total[k])
	}
	fmt.Fprintf(w, "| all | %s |\n", strings.Join(cells, " | "))
	if !withVerdicts {
		return nil
	}
	fmt.Fprintf(w, "\n| Area | Rows | Count |\n|---|---|---|\n")
	var rows strings.Builder
	for _, area := range areas {
		file := area + ".divergences.tsv"
		old, err := manifest(filepath.Join(oldDir, file))
		if err != nil {
			return err
		}
		cur, err := manifest(filepath.Join(newDir, file))
		if err != nil {
			return err
		}
		changes := verdictChanges(old, cur)
		for _, k := range slices.Sorted(maps.Keys(changes)) {
			fmt.Fprintf(w, "| %s | %s | %d |\n", area, k, changes[k])
		}
		if err := writeRows(&rows, newDir, area, rowChanges(old, cur)); err != nil {
			return err
		}
	}
	fmt.Fprintf(w, "\n| Area | Name | Reference | Before | Now | The reference's answer now |\n|---|---|---|---|---|---|\n%s", rows.String())
	return nil
}

// writeRows writes each row that moved with the answer its reference gives
// now: the evidence each reclassification is read against.
func writeRows(w io.Writer, newDir, area string, changes []rowChange) error {
	answers := map[string]map[string]json.RawMessage{}
	for _, c := range changes {
		if _, ok := answers[c.ref]; !ok && c.ref != "reference" {
			res, err := results(filepath.Join(newDir, area+"."+c.ref+".json"))
			if err != nil {
				return err
			}
			answers[c.ref] = res
		}
		answer := "—"
		if raw, ok := answers[c.ref][c.name]; ok {
			var compact bytes.Buffer
			if err := json.Compact(&compact, raw); err != nil {
				return fmt.Errorf("%s %s: %w", area, c.name, err)
			}
			// A | in a cell would split the row.
			answer = "`" + strings.ReplaceAll(compact.String(), "|", `\|`) + "`"
		}
		fmt.Fprintf(w, "| %s | %s | %s | %s | %s | %s |\n", area, c.name, c.ref, or(c.before, "—"), or(c.now, "gone"), answer)
	}
	return nil
}

// or is a class, or what stands for it where the manifest holds no row.
func or(class, absent string) string {
	if class == "" {
		return absent
	}
	return class
}
