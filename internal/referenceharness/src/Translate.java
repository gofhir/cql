import org.cqframework.cql.cql2elm.*;
import org.hl7.elm.r1.Library;
import java.io.*;
import java.nio.file.*;
import java.util.*;

/** Traduce CQL a ELM JSON con anotaciones de tipo, para poder difear
 *  lo que el traductor de referencia infiere contra lo que hace el motor. */
public class Translate {
    // 5.x logs through kotlin-logging, which prints a startup line to stdout, where
    // the JSON goes. Turned off before any class of the library loads it.
    static {
        io.github.oshai.kotlinlogging.KotlinLoggingConfiguration.INSTANCE.setLogStartupMessage(false);
    }

    public static void main(String[] args) throws Exception {
        if (args.length < 1) {
            System.err.println("uso: Translate <fichero.cql> [dir-de-librerias]");
            System.exit(2);
        }
        Path file = Paths.get(args[0]);
        Path dir = args.length > 1 ? Paths.get(args[1]) : file.getParent();

        ModelManager models = new ModelManager();
        // 5.x reads libraries through kotlinx-io: the provider takes its Path, not java.nio's.
        LibrarySourceProvider src = new DefaultLibrarySourceProvider(kotlinx.io.files.PathsJvmKt.Path(dir.toString()));
        LibraryManager libs = new LibraryManager(models);
        libs.getLibrarySourceLoader().registerProvider(src);

        libs.getCqlCompilerOptions().setOptions(
                CqlCompilerOptions.Options.EnableResultTypes,
                CqlCompilerOptions.Options.EnableLocators,
                CqlCompilerOptions.Options.EnableAnnotations);
        CqlTranslator t = CqlTranslator.fromFile(file.toString(), libs);
        if (!t.getErrors().isEmpty()) {
            // Exit without printing. A reference derived from a failed
            // translation would be recorded as authoritative and diffed
            // against, which is the one thing this must never produce.
            for (Object e : t.getErrors()) System.err.println("ERROR: " + e);
            System.exit(1);
        }
        System.out.println(t.toJson());
    }
}
