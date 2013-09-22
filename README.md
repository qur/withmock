withmock / mocktest
===================

What is this?
-------------

A pair of tools to automatically mock packages for testing with gomock

How do I install it?
--------------------

As you might expect:

    go get github.com/qur/withmock
    go get github.com/qur/withmock/mocktest

You will also need to install goimports (github.com/bradfitz/goimports), and
gomock (code.google.com/p/gomock/gomock).

How do I use it?
----------------

withmock allows an arbitrary command to be run, but will only setup the test
environment for the package in the current directory.  mocktest builds it's own
command to be run, but can test multiple packages at the same time.

To configure which packages get mocked, you just add "// mock" to the end of
your import in the test code.  Then you can use with withmock or mocktest to
actually run the test.  To use withmock you simply prepend withmock to your
normal test command, e.g.:

    withmock go test

To use mocktest you specify the packages on the command line, so to test the
current directory in the same manner as the above withmock command you can just
run:

    mocktest

However, if you wanted to run a heirarchy of test you could run:

    mocktest ./...

For more info see the documentation: http://godoc.org/github.com/qur/withmock

You can also check out the example.
