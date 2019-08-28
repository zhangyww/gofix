Go 1.11 and upper, go introduce a new module deps manage way - go module. 
When using go module, you can not import your module package with relative path.
You can only import module package with module prefix.

So, i wirte small tool to fix this. This tool can replace old relative import pattern 
with new module prefix pattern.

# VERSION and BUGFIX

v0.0.0		first gofix version. only can fix old relative import path.
v0.0.1		add -old arg to replace modulename.


# USAGE

First, you should build this project, generate a executable file.

```
cd gofix
go build -o gofix gofix.go
```

Then, make sure go.mod file exist in your going-to-fix project, 
and module name has been specified in go.mod file.

you can use **go mod init** cmd to generate a go.mod file.


Finally, running gofix, with your project path as **-dir** arg.

```
./gofix  -dir=path/to/your/project
```

If your project already uses the module import pattern, 
but you want to change the module name. Fisrt you should change module name
to newer one in go.mod file. then, run gofix witch **-old** specify the old module name

```
./gofix  -dir=path/to/your/project -old=oldModuleName
```

# NOTE!!

i can not guaratee than this tool has no bugs. 
So before using this tool, make sure you have backup your project.
This tool will override *.go file in the project. 