import org.cqframework.cql.cql2elm.*;
import org.hl7.elm.r1.Library;
import java.io.*;
import java.nio.file.*;
import java.util.*;

/** Traduce CQL a ELM JSON con anotaciones de tipo, para poder difear
 *  lo que el traductor de referencia infiere contra lo que hace el motor. */
public class Translate {
    public static void main(String[] args) throws Exception {
        if (args.length < 1) {
            System.err.println("uso: Translate <fichero.cql> [dir-de-librerias]");
            System.exit(2);
        }
        Path file = Paths.get(args[0]);
        Path dir = args.length > 1 ? Paths.get(args[1]) : file.getParent();

        ModelManager models = new ModelManager();
        LibrarySourceProvider src = new DefaultLibrarySourceProvider(dir);
        LibraryManager libs = new LibraryManager(models);
        libs.getLibrarySourceLoader().registerProvider(src);

        libs.getCqlCompilerOptions().setOptions(
                CqlCompilerOptions.Options.EnableResultTypes,
                CqlCompilerOptions.Options.EnableLocators,
                CqlCompilerOptions.Options.EnableAnnotations);
        CqlTranslator t = CqlTranslator.fromFile(file.toFile(), libs);
        if (!t.getErrors().isEmpty()) {
            for (Object e : t.getErrors()) System.err.println("ERROR: " + e);
        }
        System.out.println(t.toJson());
    }
}
