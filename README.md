withmock
========

What is this?
-------------

A tool to automatically mock packages for testing with gomock

How do I install it?
--------------------

As you might expect:

    go get github.com/qur/withmock

You will also need to install goimports (github.com/bradfitz/goimports), and
gomock (code.google.com/p/gomock/gomock).

How do I use it?
----------------

Basically, you just add "// mock" to the end of your import in the test code,
then run "go test" via the withmock tool:

    withmock go test

Check out the example for more information.
