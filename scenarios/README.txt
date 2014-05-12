basic           - a basic "does this work at all" test.

basic_stdlib    - similar to basic, but mocking a stdlib package

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

runtime         - check that we can control the mocking behaviour at runtime.
                  this means enabling and disabling mocks either across an
                  entire package, or for parts of a package on a test by test
                  basis.

shared_types    -

uses_gomock     - packages that already use gomock should only import it once.

with_deps       -

func_literals   - originally we would replace function literals with a function
                  that would panic if you actually called it - since it was the
                  easiest way to know what to put there.  since implementing
                  runtime control this is no longer appropriate, and we need to
                  make sure that the original function body is available.

excludes        - make sure that when we specify that a package is excluded from
                  mocking that it actually does get excluded.

embedding       - the mocking should handle embedded types and the methods
                  implemented by those types pretty much as expected.

issue16         - We weren't correctly generating code for constants that used
                  a explicit type from another package.

issue17         - We were trying to rewrite packages with non-Go code in them
                  for mocking.  However, this didn't actually work because we
                  were only rewriting the Go code.  This scenario includes
                  packages with asm code and packages with C code.  We then try
                  and use one mocked and one unmocked version of each kind of
                  package.  We can't rewrite these types of packages for runtime
                  control - but we should be able to mock them using the old
                  approach of _just_ writing out mocked code (provided that we
                  ignore the non-Go files, otherwise the toolchain will try and
                  include those too).

nongocode       - Make sure that we can test packages that include non-go code.

issue18         - If an imported package includes non-Go code, then we don't
                  adds any package it imports into the list of packages to be
                  installed.  This then results in go test failing to compile
                  the test binary.

issue19         - If two packages are dot-imported then we will get name
                  conflicts over the MOCK and EXPECT functions that we add to
                  all packages.

struct_tags     - Tags on struct fields were not being copied to the generated
                  code, which breaks things like JSON encode/decode that use
                  tags.

missing         - The error message produced when a package is missing is really
                  cryptic, we should be detecting missing packages, and using an
                  explicit error message.

issue23         - If a package is imported by two sets of code under test, and
                  the first marks it to be mocked then the mocking will also be
                  enabled for the second.  If the second package didn't ask for
                  mocking enabled, it shouldn't be.

build_constraints - make sure that build constraints are respected.  This
                  includes using a mocked version of the os package, which uses
                  a combination of explicit and implicit build contraints. In
                  particular, build constraints that aren't the first comment
                  didn't work.

issue24         - If withmock/mocktest are used to test a package outside of
                  GOPATH, it fails.

has_init	- If a package we are processing has init methods, then we
		  need to make sure they are called when mocking is disabled -
		  otherwise the package will not behave correctly.

issue25		- If a package that is being mocked imports a non-existant
                  package from a file that should be excluded by a build tag
                  (e.g. using a package that only exists in a newer version of
                  Go), then we will error out when trying to process that file
                  as we will fail to find the import.  We should probably assume
                  that the file won't be compiled - but try and setup a nice
                  error in case it is ...

issue27         - If we create two instances of the same type of interface Mock,
                  then they should be independant - and expectations registered
                  against one instance should not be satisified by calls against
                  the other (which they are when using struct{} as the
                  underlying type for the instance mock).

issue28         - If a package embeds a C library, then that library will not be
                  setup inside the working directory - and so the package will
                  not compile.

issue31         - When we generate a mock instance for an interface we don't
                  handle varidic methods correctly.

issue32         - Unable to mock time package

issue33         - Unable to mock bytes package

issue34         - Unable to mock os/signal package

issue35         - Unable to mock net package

readme          - Add new mechanism for mocking packages by replacing with
                  alternative sources (i.e. manually written mocks for packages
                  we can't automatically mock).  This also provides a workaround
                  for things that we can't mock dues to bugs etc.

stdlib_cross    - Currently we can mock stdlib packages, but you can't pass the
                  values from a mocked stdlib package to another package because
                  that package is using the real version (and the types are
                  actually different).

other_c_code    - When a package contains a sub directory with a .go file in it
                  we will try an install it.  However, if we only included the
                  path because we saw it as a subdirectory then we shouldn't try
                  to install it.  We should only install packages that are
                  imported as part of the import chain from the test code.

protobuf        - We weren't able to process the goprotobuf package reliably.
                  Because there are two definitions of a type with build
                  contraints selecting the one to actually use we would flip
                  between literal depending on the processing order.

var_slice       - We weren't handling slice expressions in exprString.
