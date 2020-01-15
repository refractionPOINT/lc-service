#! /bin/sh

export SPHINX_APIDOC_OPTIONS="members,no-undoc-members,show-inheritance"

sphinx-apidoc -f -o ./docs/ limacharlie

cd docs ; make html ; cd ..