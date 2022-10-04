Use command "go build ./main.go" in the project directory for building a project.

The parameters for launching the application are specified in "config.env" file in a project directory.
Port, number of nodes and key space size must be specified before launch.

For executing an application use command "<PATH_TO_PROJECT_DIRECTORY>/main <predecessorIp> <successorIp> <nodeNumber>"

This application should be executed on every machine we would like to use in the cluster.

A script for quick launching of the system is TBD.

Here is an example of launching the 16-nodes system on compute-10-... with a list of commands:

ssh -f compute-10-0 <PROJECT DIRECTORY>/main 10.1.2.121 10.1.2.197 0
ssh -f compute-10-1 <PROJECT DIRECTORY>/main 10.1.2.233 10.1.2.184 1
ssh -f compute-10-2 <PROJECT DIRECTORY>/main 10.1.2.197 10.1.2.183 2
ssh -f compute-10-3 <PROJECT DIRECTORY>/main 10.1.2.184 10.1.2.182 3
ssh -f compute-10-4 <PROJECT DIRECTORY>/main 10.1.2.183 10.1.2.181 4
ssh -f compute-10-5 <PROJECT DIRECTORY>/main 10.1.2.182 10.1.2.180 5
ssh -f compute-10-6 <PROJECT DIRECTORY>/main 10.1.2.181 10.1.2.179 6
ssh -f compute-10-7 <PROJECT DIRECTORY>/main 10.1.2.180 10.1.2.178 7
ssh -f compute-10-8 <PROJECT DIRECTORY>/main10.1.2.179 10.1.2.177 8
ssh -f compute-10-9 <PROJECT DIRECTORY>/main 10.1.2.178 10.1.2.161 9
ssh -f compute-10-10 <PROJECT DIRECTORY>/main 10.1.2.177 10.1.2.143 10
ssh -f compute-10-11 <PROJECT DIRECTORY>/main 10.1.2.161 10.1.2.142 11
ssh -f compute-10-12 <PROJECT DIRECTORY>/main 10.1.2.143 10.1.2.123 12
ssh -f compute-10-13 <PROJECT DIRECTORY>/main 10.1.2.142 10.1.2.122 13
ssh -f compute-10-14 <PROJECT DIRECTORY>/main 10.1.2.123 10.1.2.121 14
ssh -f compute-10-15 <PROJECT DIRECTORY>/main 10.1.2.122 10.1.2.233 15