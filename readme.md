Use command "go build ./main.go" in the project directory for building a project.

The parameters for launching the application are specified in "config.env" file in a project directory.
Port and key space size must be specified before launch.

For executing an application use command "<PATH_TO_PROJECT_DIRECTORY>/main"

This application should be executed on every machine we would like to use in the cluster.

start.sh is the script for quick launching of the system on a set of UIT cluster nodes.
The amount of nodes needed to be set up should be given as a script paremeter.
For example, to set the system on 20 nodes (first 20 nodes from the script addresses array), use command "<PATH_TO_PROJECT_DIRECTORY>/start.sh 20"

join.sh is the script for quick setting up the network.
The amount of nodes should and a port should be given as a script paremeters.
For example, to set up the network of 20 nodes on port 58346 (first 20 nodes from the script addresses array), use command "<PATH_TO_PROJECT_DIRECTORY>/join.sh 20 58346"

!IMPORTANT! An amount of nodes in start.sh and join.sh scripts should be the same for the correct work.