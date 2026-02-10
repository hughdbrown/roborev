/hdb:design "Design a tool for managing the sqlite database of email data. Its core purposes are:
  - to control which version of the data is active (by soft-linking directories to ~/.msgvault)
  - to maintain multiple versions of the data in multiple directories, with only one copy visible to msgvault at a time
  - to create expendable subsets of data for development by copying emails from a source data set to a copy data set
  The tool should be created in golang. The tool should live in `tools/`.
  Switching between different data sets is achieved by executing a soft-link. In MacOS and linux, this is `ln -s`. In Windows, it is
  something else.
  The verbs supported by the tool are:
  - mount-data: given a directory that has msgvault data in it, point `~.msgvault` at this directory
  arguments: `--dataset <dataset-name>` which will be in the directory `~/.msgvault-<dataset-name>`
  - init-dev-data: set up the use of soft link pointers around `.msgvault`. If there are no soft link pointers, move `~/.msgvault` to
  `~/.msgvault-gold` and soft link `~/.msgvault` to `~/.msgvault-gold`
  - exit-dev-data: Undo the effects of init-dev-data by deleting the link of `~/.msgvault` to some other directory and renaming
  `~/.msgvault-gold` to `~/.msgvault`
  - new-data: create an excerpt of the data in dataset `src` (in directory `~/.msgvault-<src>` if src is provided and in
  `~/.msgvault` otherwise) and place data into dataset `dst` (in directory `~/.msgvault-<dst>`)
  arguments: --src <src> optional; --dst <dst>; --rows <row-count>"

