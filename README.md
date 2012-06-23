External images on LinuxFr.org
==============================

Our users can use images from external domains on LinuxFr.org.
This component is a reverse-proxy / cache for these images.

Th main benefits of using a proxy instead of linking directly the images are:

- no flood: images can be hosted on small servers that are not able to handle
  all the traffic from LinuxFr.org, so we avoid to flood them ;
- history: even if a server is taken down, we are able to keep serving images
  that are already used on our pages ;
- security: on the HTTPS pages, we won't include images from other domains
  that are available only in HTTP, so it prevents browsers from displaying
  warning about unsafe pages ;
- privacy: the users won't connect to the external domains, so their IP
  addresses won't be logged on these servers.


How to use it?
--------------

[Install Go 1](http://golang.org/doc/install) and don't forget to set `$GOROOT`

    $ go get github.com/bmizerany/pat
    $ go get github.com/nono/img-LinuxFr.org
    $ img [-p port] [-s secret]


Why don't you use camo?
-----------------------

Github has developed a similar service, camo, for their own usage.
I used it as a source of inspiration but I prefered to redevelop a new service
instead of using it for several reasons:

- it lacks some feature and particulary caching!
- it runs with a legacy version of node.js, which is not very friendly for our
  sysadmins.
- I plan to add extend it for other usages in the future, and I prefer coding
  in golang than in nodejs.


See also
--------

* [Git repository](http://github.com/nono/img-LinuxFr.org)
* [Camo](https://github.com/atmos/camo)


Copyright
---------

The code is licensed as GNU AGPLv3. See the LICENSE file for the full license.

♡2012 by Bruno Michel. Copying is an act of love. Please copy and share.