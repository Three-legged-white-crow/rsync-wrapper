DATE=$(shell date -u '+%Y-%m-%d_%H-%M-%S')
GIT_VERSION=$(shell git rev-parse HEAD)

# collect all static bin
bin-collect:
	mv transporter/cmd/checksum/checksum rsync-wrapper_bin/
	mv transporter/cmd/create_wrapper/create-wrapper rsync-wrapper_bin/
	mv transporter/cmd/mv_wrapper/mv-wrapper rsync-wrapper_bin/
	mv transporter/cmd/rm_wrapper/rm-wrapper rsync-wrapper_bin/
	mv transporter/cmd/stat_wrapper/stat-wrapper rsync-wrapper_bin/
	mv transporter/cmd/copy/copy rsync-wrapper_bin/
	mv transporter/cmd/copy_cleaner/copy-cleaner rsync-wrapper_bin/
	mv transporter/cmd/copylist/copylist rsync-wrapper_bin/
	mv rsync/rsync-static rsync-wrapper_bin/

bin-compress:
	tar -Jcvf "rsync-wrapper_bin_${DATE}_${GIT_VERSION}.tar.xz" rsync-wrapper_bin/

collect-clean:
	rm -rf rsync-wrapper_bin/*


docker:
	docker build -t "rsync-wrapper:test_${DATE}_${GIT_VERSION}" --rm --no-cache .
