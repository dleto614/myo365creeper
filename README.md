# o365creeper: My variant in Go

So the original o365creeper is here: https://github.com/LMGsec/o365creeper

And it is in python2. Doesn't seem like it will be ported to python3 so I decided on a whim to port it over to Go and write it the way I want the tool to act like. (It is what I did with prips and you can find that repository here: https://github.com/dleto614/myprips)

----

You can compile this program by running:

```bash
git clone https://github.com/dleto614/myo365creeper
cd myo365creeper && go build
```

----

Usage:

```bash
$ ./o365creeper -h
Usage of ./o365creeper:
  -e    Only write valid email addresses to the output file.
  -i string
        Specify input file with email addresses.
  -o string
        Specify the output file to write results into.
```

Fairly simple tool to use. You can give a list of emails via `-i` and if you just want to only get the valid emails, you can specify `-e`. If not, by default it outputs:

\- Username (email)  
\- Display Name  
\- Valid (true or false)  
\- is_unmanaged (true or false)  
\- throttle_status (0 or 1 not sure if can be any other number)  
\- is_signup_disallowed  

---

There seems to be some weird throttling going on, but to use:

```bash
$ ./o365creeper -i emails.txt -o valid-emails.txt -e # Only write to file valid emails
$ ./o365creeper -i emails.txt -o valid-emails.json # Write full json output
$ ./o365creeper -i emails.txt -e # Output to STDOUT the valid emails
$ ./o365creeper -i emails.txt # Output to STDOUT the full json output
```

Regarding the weird throttling, in my testing, I would get these kind of results:

```bash
...
  "IfExistsResult": 0,
  "IsUnmanaged": false,
  "ThrottleStatus": 1,
  "Credentials": {
...
```

For some reason when `ThrottleStatus` is set to 1, everything for `IfExistsResult` is set to 0 no matter what email is inputted. If it was actually throttled, `IfExistsResult` should return 2. At least from what I read, but I could be wrong.

I don't know why Microsoft does this, but it is very counterproductive.