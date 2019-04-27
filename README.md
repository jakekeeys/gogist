# gogist
A CLI tool for creating and managing github's gists

## Building from source
* Install the go runtime

#### Using the go toolchain
* Compile and install binary
```
go install github.com/jakekeeys/gogist
```

#### Using git
* Clone the repository 
```
git clone git@github.com:jakekeeys/gogist.git
```
* Compile the binary by running the following from the repo
```
go build .
```
* Move the binary onto your path

## Usage

### Login
Login utilises github's oauth flow in order to provision a token for the application.
If successful the obtained token will be saved to `$HOME/.gogist` 
The generated personal token will only have access to gist resources and can be found under `Personal access tokens` in `Developer settngs` and will be described as `$HOSTNAME/gogist`  

You can opt to provision the token yourself and write it to `$HOME/.gogist` for consumption by the application.
```
echo -n "<token>" > ~/.gogist
```

#### Examples
Login without multi factor auth
```
gogist login -u <username> -p <password>
```

Login with multi factor auth
```
gogist login -u <username> -p <password> -o <mfa_token>
```

### New
By default new will read from stdin and create a gist with a single file containing the text consumed from it.

You can opt to name the file by passing the `-n <name>` argument but this will be ignored when passing the `--file` `--dir` or `--glob` arguments and the original filenames will instead be used.

You can opt to pass a description using the `-d <desc>` argument.

And you can opt to make the gist public by passing the `-p` argument

Rather than sourcing the content from stdin you can directly consume files by passing one or more instances of the `--file <path_to_file>` argument.
You may also pass one or more instances of the `--dir <path_to_dir>` argument and all files in the directory will be included in the gist.
You can also pass one or more instances of the `--glob <glob>` argument and all matching files will be included in the gist.
You can also use any combination of the above but note that the paths are stripped from the matching files and an error will be thrown when 2 or more files have the same name

#### Examples

Create a gist using stdin
```
cat main.go | gogist new
```

Create a gist using stdin and name the file `main.go` 
```
cat main.go | gogist new -n main.go
```

Create a gist using stdin and name the file main.go with a description of `the gogist source`
```
cat main.go | gogist new -n main.go --desc="the gogist source"
```

Create a gist from the file main.go
```
gogist new --file main.go
```

Create a gist for all files in the current directory with a description of `the gogist source`
```
gogist new --dir . --desc="the gogist source"
```

Create a gist for all go files in the gogist folder and including `README.md` and `LICENSE` files with a description of `the gogist source`
```
gogist new --glob="./gogist/*.go" --file ./gogist/README.md --file ./gogist/LICENSE -desc="the gogist source
```

### List
By default list will return URLs for all of the gists for the authenticated user
If the `-u <username>` argument is passed and matches the authenticated user urls for all public and private gists for the user are returned
If the `-u <username>` argument is passed and does not match the authenticated user urls for all public gists are returned for the user

#### Examples
List all public gists for the authenticated user
```
gogist list
```

List all gists for the user `jakekeeys`
```
gogist list -u jakekeeys
```
