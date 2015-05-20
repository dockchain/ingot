FROM busybox
ADD ./ingot /ingot
ADD ./sample.pem /sample.pem
CMD ["/ingot"]
