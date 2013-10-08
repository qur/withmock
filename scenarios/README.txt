basic           - a basic "does this work at all" test.

interface       - do we generate a Mock that implements any interface in the
                  package?

issue10         - we should handle complex types in var statements.  originally
                  we only handle simple expressions (e.g. "int") for types in
                  var statements, and more complicate expressions (e.g.
                  "*os.File") would cause us to print the value of the Expr
                  value instead of a sensible string representation.

issue11         - do dependancies use the same package as the code under test?
                  originally we would generated separate (mocked) code, only
                  used by the code under test.  this meant that it was
                  impossible to pass types declared by that package to an
                  imported package, as that package would be using the unmocked
                  version - and therefore the type would actually be different.

issue8          - do we correctly generate code using "chan foo" (originally we
                  would write "<-chan<- foo" for this)?

issue9          - we shouldn't have a problem with code that is trying to use a
                  package marked to be mocked from code other than the code
                  under test.

multiple_pkgs   - can we test multiple packages with a single command

new_methods     - do we generate NewXXX methods for private types, so that we
                  can create mocks that might be needed to satisfy interfaces
                  (e.g. a private type where the package defines a NewXXX
                  function that returns an io.Writer).

no_pkgs         - do we exit sensibly when presented with no packages to test?

runtime         -

shared_types    -

uses_gomock     - packages that already use gomock should only import it once.

with_deps       -
