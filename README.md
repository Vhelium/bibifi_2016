# Team Pathetic

## Links
Specifications: https://builditbreakit.org/static/doc/fall2016/index.html  
Mainpage: https://builditbreakit.org/details  
VM: https://umd.app.box.com/s/8lj3908flgw9q56de90o1ux4r7nlzbbw

## Testing
server build: `make`  
server run: `make run`  
client run: `cat test/test001.txt | nc localhost 6666`  
  
or from tests/: `./run.py ../build/server test1.json`

## Possible Attacks
* Non-termating program (timeout)
* Invalid string (that doesn't match the specs)

## How To: Extend Parser
Assuming you want to add support for a command called *FooBar*.  
In *parser.go*:
* Add a struct `CmdFooBar`
* Create a function `parseCmdFooBar` which returns a status code and the parsed `struct CmdFooBar`
* In `parseLine(..)` extend the token loop to add the first token of your command, e.g. `KV_FOREACH`

## How To: Extend Executor
Assuming you want to add support for a command called *FooBar*.  
In *executor.go*:
* Create function `func (cmd CmdFooBar) execute(enc *ProgramEnv) int` which executes the given command and returns a status code
