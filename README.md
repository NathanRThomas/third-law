# third-law
Monitors ip:port and executes a command if it fails.
Super simple, but sometimes that's all you need.  Allows you to switch a floating ip, or to swap an nginx file, or really anything you want.

You can manually cause a swap by doing a 
`kill -10 pid`
This service is monitoring for those system interrupts.

