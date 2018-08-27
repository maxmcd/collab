#!/usr/bin/env ruby

message = ENV["DEFAULT_MSG"] || "commit from #{`git config --get user.name`}"

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
        # It seems that merge conflicts are just written to the files. This 
        # seems ok, the users can just fix the conflicts and the files will
        # save and update.
    end

    if `git status --porcelain` != ""
        puts "Found changes, pushing!"
        `git add --all`
        `git commit -m "#{message}"`
        `git push origin #{branch}`
    end
    sleep 1
end
