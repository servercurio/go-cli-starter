package version

//go:generate touch commit.txt
//go:generate touch version.txt
//go:generate chmod +x lookup_commit_hash.sh
//go:generate ./lookup_commit_hash.sh
