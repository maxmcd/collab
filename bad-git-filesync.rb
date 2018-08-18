#!/usr/bin/env ruby

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
