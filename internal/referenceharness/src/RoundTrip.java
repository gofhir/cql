import ca.uhn.fhir.context.FhirContext;
import ca.uhn.fhir.parser.IParser;
import com.fasterxml.jackson.databind.JsonNode;
import com.fasterxml.jackson.databind.ObjectMapper;
import org.hl7.fhir.instance.model.api.IBaseResource;
import org.opencds.cqf.cql.engine.runtime.ClassInstance;
import java.nio.file.Files;
import java.nio.file.Path;
import java.nio.file.Paths;
import java.util.ArrayList;
import java.util.List;
import java.util.stream.Stream;

/** Holds Evaluate.toHapi, the way the harness writes a FHIR value 5.x hands it,
 *  to the value: each resource read is turned into the engine's value as its
 *  retrieve providers do (toCqlValue), and the HAPI object toHapi rebuilds from
 *  it has to give the engine that same value back. Over the resources of a data
 *  file (a JSON object of resources by type, as fhir-data.json) and over a
 *  directory of resources, one per file, as the package hl7.fhir.r4.examples
 *  holds them.
 *
 *  It is the engine's value that has to come back, not the resource's text:
 *  what the engine is not handed, the harness cannot write. HAPI reads a
 *  markdown without its trailing spaces, and the engine keeps a DateTime's
 *  offset, not how it was written (+00:00 or Z), so toHapi writes those as
 *  the engine has them. It writes how many resources come back as the same
 *  JSON besides, to see that this is all that differs (plan 007).
 *
 *  It writes how many it compared, and exits with 1 if one differs or if it
 *  compared none: a check that compared nothing would pass. */
public class RoundTrip {
    /** The resources whose narrative does not come back, each for a loss that is
     *  not the harness's, with what loses it. Each has to differ still, a loss
     *  that is gone is taken off; and only in the narrative's whitespace:
     *  without the narrative the value has to come back, and the narrative's
     *  text, whitespace collapsed, too, so that the exception hides nothing else. */
    static final java.util.Map<String, String> KNOWN = java.util.Map.of(
            // org.hl7.fhir.utilities 6.9.4.1 and 7.0.3: XhtmlParser and
            // XhtmlComposer add two spaces before a comment each time a
            // narrative is read and written again; toHapi has to read it.
            // hapifhir/org.hl7.fhir.core#2785.
            "PlanDefinition-KDN5.json", "a narrative's comment, reindented by org.hl7.fhir.utilities");

    public static void main(String[] args) throws Exception {
        if (args.length == 0) {
            System.err.println("usage: RoundTrip <data.json | directory>...");
            System.exit(2);
        }
        IParser parser = FhirContext.forR4Cached().newJsonParser();
        ObjectMapper json = new ObjectMapper();
        int compared = 0, sameJSON = 0;
        List<String> differ = new ArrayList<>();
        for (String arg : args) {
            for (Resource r : resources(Paths.get(arg), json)) {
                IBaseResource resource;
                try {
                    resource = parser.parseResource(r.json);
                } catch (Exception e) {
                    // Not a resource HAPI reads: not one the engine would be handed.
                    continue;
                }
                ClassInstance value = (ClassInstance) Evaluate.FHIR.toCqlValue(resource, false);
                String want = value.toString();
                boolean known = KNOWN.containsKey(r.name);
                String got;
                boolean restSame = false;
                try {
                    IBaseResource rebuilt = (IBaseResource) Evaluate.toHapi(value);
                    got = Evaluate.FHIR.toCqlValue(rebuilt, false).toString();
                    if (known) restSame = withoutNarrative(rebuilt) != null
                            && withoutNarrative(rebuilt).equals(withoutNarrative(resource))
                            && narrativeText(rebuilt).equals(narrativeText(resource));
                    if (parser.encodeResourceToString(rebuilt).equals(parser.encodeResourceToString(resource))) sameJSON++;
                } catch (Exception | Evaluate.EncodingError e) {
                    got = "error: " + e;
                }
                compared++;
                if (known && want.equals(got)) differ.add(r.name + ": the known loss is gone (" + KNOWN.get(r.name) + "); take it off");
                if (known && !want.equals(got) && !restSame) differ.add(r.name + ": more than the known loss differs ("
                        + KNOWN.get(r.name) + ")\n  want " + want + "\n  got  " + got);
                if (known && !want.equals(got) && restSame) System.out.println(r.name + ": known loss, " + KNOWN.get(r.name));
                if (!known && !want.equals(got)) differ.add(r.name + "\n  want " + want + "\n  got  " + got);
            }
        }
        for (String d : differ) System.out.println(d);
        System.out.println(compared + " resources compared, " + differ.size() + " differ; "
                + sameJSON + " come back as the same JSON");
        if (compared == 0 || !differ.isEmpty()) System.exit(1);
    }

    /** The engine's value of a resource with its narrative's XHTML taken off;
     *  null for one with no narrative, which no known loss can be. */
    static String withoutNarrative(IBaseResource resource) {
        if (!(resource instanceof org.hl7.fhir.r4.model.DomainResource d)) return null;
        org.hl7.fhir.r4.model.DomainResource bare = d.copy();
        // Only the XHTML: the narrative's status is the value's, and has to come back.
        if (bare.hasText()) bare.getText().setDiv(null);
        return Evaluate.FHIR.toCqlValue(bare, false).toString();
    }

    /** A resource's narrative with its whitespace collapsed: hapifhir/org.hl7.fhir.core#2785
     *  changes only the whitespace, and any other change is not that loss. */
    static String narrativeText(IBaseResource resource) {
        org.hl7.fhir.r4.model.DomainResource d = (org.hl7.fhir.r4.model.DomainResource) resource;
        return d.getText().getDivAsString().replaceAll("\\s+", " ");
    }

    record Resource(String name, String json) {}

    static List<Resource> resources(Path path, ObjectMapper json) throws Exception {
        List<Resource> out = new ArrayList<>();
        if (Files.isDirectory(path)) {
            try (Stream<Path> files = Files.list(path)) {
                for (Path f : files.filter(f -> f.toString().endsWith(".json")).sorted().toList()) {
                    JsonNode n = json.readTree(Files.readString(f));
                    if (n.has("resourceType")) out.add(new Resource(f.getFileName().toString(), n.toString()));
                }
            }
            return out;
        }
        JsonNode root = json.readTree(Files.readString(path));
        root.fields().forEachRemaining(e -> {
            List<JsonNode> items = new ArrayList<>();
            if (e.getValue().isArray()) e.getValue().forEach(items::add); else items.add(e.getValue());
            for (JsonNode item : items) out.add(new Resource(path.getFileName() + " " + e.getKey(), item.toString()));
        });
        return out;
    }
}
