handbraked
==========

Watches the specified directory for new files (or ideally, symlinks to files), 
then re-encodes them using handbrake and deletes the original file.

Can be used in conjunction with https://github.com/tangerilli/podcasts-go. Workflow
is:

1. Symlink movie you want to watch on iPad into watched directory
2. Movie is re-encoded into correct format and placed into output directory
3. podcasts-go is using output directory to serve podcasts XML, which iPad processes and uses to download movie