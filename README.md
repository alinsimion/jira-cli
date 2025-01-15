# jira-cli
A command line interface for interacting with JIRA

## Platform specific executables 

[Windows](https://github.com/alinsimion/jira-cli/releases/download/NewRelease/myprogram-windows-amd64)
[Linux](https://github.com/alinsimion/jira-cli/releases/download/NewRelease/myprogram-linux-amd64)
[MacOS](https://github.com/alinsimion/jira-cli/releases/download/NewRelease/myprogram-darwin-amd64)

## Releases

![GitHub release (latest by date)](https://img.shields.io/github/v/release/alinsimion/jira-cli?style=flat-square)

### Currently supports:
- Logging work
- Viewing issues

Usage:
```
jira-cli [flags]
jira-cli [command]
```

Available Commands:
```
dumpenv     dumps empty template .env file in cwd
help        Help about any command
list        lists your issues
logwork     helps with logging work
```

To make use of this tool, you need to create a `.env` file in the root of the project with the following content:
```
JIRA_API_KEY=[your_jira_api_key]
JIRA_ENDPOINT=[your_jira_endpoint]
JIRA_USERNAME=[your_jira_username]
```

To get your JIRA API key, follow the instructions [here](https://support.atlassian.com/atlassian-account/docs/manage-api-tokens-for-your-atlassian-account/).

## Examples

### 1. Logging work
```
logwork -t 6 -i GAIA-1232 -d 12/07/2024

logwork -t 6 -i GAIA-1232                                       # this will log work today

logwork -t 6 -i GAIA-1232 -p week                               # this will log work for the week in progress until today (inclusive)

logwork -t 6 -i GAIA-1232 -p month                              # this will log work for the month in progress until today (inclusive)

logwork -t 6 -i GAIA-1232 -p lastmonth -m "I did some work"     # this will log 6h of work for the last month on the issue GAIA-1232 with the message "I did some work" for each entry

```

### 2. Listing issues
```
list -p GAIA    # lists all your issues in the project GAIA
```
