# collab

Thoughts on three solutions:

#### Git

It seems the simplest solution to implement would just be a misuse of git. Set up a git repo, push the initial files, immediately push any future changes in response to filesystem events, and continually poll for changes. Failed merges would stop the world, but frequent committing would likely minimize instances of failed merges. I'm fairly certain that each git commit contains the entire version of each modified file, so this solution doesn't help us with our bandwidth concern (our frequent polling interview doesn't help either).

[Here's](./bad-git-filesync.rb) a ruby version of this solution:
```ruby
if `git status` == ""
    puts "Not a git repo, exiting"
    exit!
end

if `git remote` == ""
    puts "No available remote repo, exiting"
    exit!
end

while true
    branch = `git symbolic-ref --short HEAD`.strip!
    if `git fetch && git rev-list HEAD...origin/#{branch} --count`.strip! != '0'
        puts "Found remote changes, pulling!"
        `git pull origin #{branch}`
    end

    if `git status --porcelain` != ""
        puts "Found changes, pushing!"
        `git add --all`
        `git commit -m "not useful"`
        `git push origin #{branch}`
    end
    sleep 1
end
```

#### P2P

Having worked with WebRTC recently I initially wanted to set up a pure P2P solution using a STUN server for NAT traversal and then handle all of the file syncing over UDP. I stopped exploring this for two reasons:

1. [natty](https://github.com/getlantern/go-natty) seemed like a good candidate for establishing the connection. In looking into this I don't think there's an easy way to handle the SDP exchange without a signaling server. Was considering asking the user to pass connection details through Slack (not entirely sure if that would work).

2. Writing a UDP-based file sync protocol from scratch seems like a bit much. No centralized server also means any bandwidth-saving measures would be even more complicated.

#### Dropbox

I've had [some experience](https://github.com/golangbox/gobox) writing a dropbox clone before so decided to go with this strategy. General setup would be:

 - Client library that can host a directory or pull down a shared directory
 - Files are split into 4mb chunks before being sent to the server
 - Central blob store stores chunks as their sha256 hashes
 - Websocket connection shares file metadata and chunk descriptions

Dropbox generally ignores opportunities to merge files and just creates two copies of a conflicted file. Maybe I'll try and merge text files but use the conflicting file strategy otherwise.

## Implementation

#### Structure

There is a server and a client binary. The server handles blob storage and event passing, it also keeps a master of the file tree for the initial download. The client starts hosting a directory with `collab serve shared-key`. The entire directory is crawled, indexed, and uploaded to the server in 4mb (or less) hashed chunks. The directory server then tells another client to start receiving with `collab receive shared-key`. The other client pulls down the listing of all available files and downloads their chunks, reassembling them as a replicated filesystem. From there any filesystem event is sent as a websocket message to the server and listeners are notified of changes. Listeners then pull down any file chunks they might need, or make any other necessary filesystem changes.

Current limitations/bugs:

 - Only really works at the moment if both users start their sessions simultaneously
 - Checking the contents of a file triggers a WRITE filesystem event, which then ends up checking itself a few times. Haven't found a way to simply mitigate this
 - Gzip could be easily added to file chunk requests, but it is currently not
 - The servers file tree isn't currently updated when it changes
 - Deletion is untested
 - Merge conflicts are not handled


#### Running

```bash
docker-compose up --build
# server and clients are now running

# in another terminal window:
docker-compose exec sender bash
# you can now make changes to the sender filesystem
# this directory (/opt/) is the shared directory

# in another terminal window:
docker-compose exec receiver bash
# cd into /opt/foo
# this is the copy of the hosts filesystem
```
