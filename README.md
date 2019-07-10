# Overview

This is an example of how to create a filesystem using memory-backed files. It implements all of the mechanics of a minimal, browseable, read-only filesystem.


# Example Output

User console:

```
$ find /mnt/loop
/mnt/loop
/mnt/loop/subdirectory1
/mnt/loop/subdirectory1/file4
/mnt/loop/subdirectory1/file5
/mnt/loop/subdirectory1/file6
/mnt/loop/file1
/mnt/loop/file2
/mnt/loop/file3

$ ls -il /mnt/loop
total 0
  11 -rw-r--r-- 0 dustin dustin 16 Jul 10 14:19 file1
  22 -rw-r--r-- 0 dustin dustin 16 Jul 10 14:19 file2
  33 -rw-r--r-- 0 dustin dustin 16 Jul 10 14:19 file3
1002 drwxr-xr-x 0 dustin dustin 10 Jul 10 14:19 subdirectory1

$ cat /mnt/loop/file1
test content 1

$ cat /mnt/loop/subdirectory1/file4
test content 4

$ sudo umount /mnt/loop 
```

Server console:

```
$ go run main.go 
Unmount to terminate.

Opendir: [/]
Readdir: [/]
Lookup: [/subdirectory1]
Lookup: [/file1]
Lookup: [/file2]
Lookup: [/file3]
Opendir: [/subdirectory1]
Readdir: [/subdirectory1]
Lookup: [/subdirectory1/file4]
Lookup: [/subdirectory1/file5]
Lookup: [/subdirectory1/file6]
Opendir: [/]
Readdir: [/]
Lookup: [/subdirectory1]
Lookup: [/file1]
Lookup: [/file2]
Lookup: [/file3]
Lookup: [/file1]
Open (00000000000000001000000000000000): [/file1]
Lookup: [/file1]
Lookup: [/file1]
Lookup: [/subdirectory1]
Lookup: [/subdirectory1/file4]
Open (00000000000000001000000000000000): [/subdirectory1/file4]
Lookup: [/subdirectory1/file4]
Lookup: [/subdirectory1/file4]

$
```
