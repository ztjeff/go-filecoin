package network

import (
	"bytes"
	"io/ioutil"
	"math/rand"
	"path/filepath"

	"gx/ipfs/QmXWZCd8jfaHmt4UDSnjKmGcrQMw95bDGWqEeVLVJjoANX/go-ipfs-files"
)

func RandomFile(filesdir, tmpdir string) (files.File, error) {
	/*
	   f, err := RandomFileTestDir(filesdir)
	   if err == nil {
	           return f, nil
	   }
	*/
	return RandomFileText()
}

func RandomFileTestDir(dir string) (string, error) {
	// check if there is a testdir locally
	fs, err := ioutil.ReadDir(dir)
	if err != nil {
		return "", err // failed to read testfiles
	}

	// random entry
	fi := fs[rand.Intn(len(fs))]
	fp := filepath.Join(dir, fi.Name())
	return fp, nil
}

func RandomFileText() (files.File, error) {
	rf := randomFiles[rand.Intn(len(randomFiles))]

	r := bytes.NewReader([]byte(rf))

	return files.NewReaderFile(r), nil
}

var randomFiles = []string{
	randomFile1,
	randomFile2,
	randomFile3,
}

const randomFile1 = `
▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄
▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄
▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄
▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄
▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄
▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄
▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄
▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄
▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄
▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄
▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄
▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄
▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄
▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄
▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄
▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄
▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄
▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄
▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄
▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄
▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄
▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄
`

const randomFile2 = `

███████╗██╗██╗     ███████╗ ██████╗ ██████╗ ██╗███╗   ██╗
██╔════╝██║██║     ██╔════╝██╔════╝██╔═══██╗██║████╗  ██║
█████╗  ██║██║     █████╗  ██║     ██║   ██║██║██╔██╗ ██║
██╔══╝  ██║██║     ██╔══╝  ██║     ██║   ██║██║██║╚██╗██║
██║     ██║███████╗███████╗╚██████╗╚██████╔╝██║██║ ╚████║
╚═╝     ╚═╝╚══════╝╚══════╝ ╚═════╝ ╚═════╝ ╚═╝╚═╝  ╚═══╝

`

const randomFile3 = `
A Declaration of the Independence of Cyberspace
by John Perry Barlow

Governments of the Industrial World, you weary giants of flesh and steel, I
come from Cyberspace, the new home of Mind. On behalf of the future, I ask you
of the past to leave us alone. You are not welcome among us. You have no
sovereignty where we gather.

We have no elected government, nor are we likely to have one, so I address you
with no greater authority than that with which liberty itself always speaks. I
declare the global social space we are building to be naturally independent of
the tyrannies you seek to impose on us. You have no moral right to rule us nor
do you possess any methods of enforcement we have true reason to fear.

Governments derive their just powers from the consent of the governed. You
have neither solicited nor received ours. We did not invite you. You do not
know us, nor do you know our world. Cyberspace does not lie within your
borders. Do not think that you can build it, as though it were a public
construction project. You cannot. It is an act of nature and it grows itself
through our collective actions.

You have not engaged in our great and gathering conversation, nor did you
create the wealth of our marketplaces. You do not know our culture, our
ethics, or the unwritten codes that already provide our society more order
than could be obtained by any of your impositions.

You claim there are problems among us that you need to solve. You use this
claim as an excuse to invade our precincts. Many of these problems don't
exist. Where there are real conflicts, where there are wrongs, we will
identify them and address them by our means. We are forming our own Social
Contract. This governance will arise according to the conditions of our world,
not yours. Our world is different.

Cyberspace consists of transactions, relationships, and thought itself,
arrayed like a standing wave in the web of our communications. Ours is a world
that is both everywhere and nowhere, but it is not where bodies live.

We are creating a world that all may enter without privilege or prejudice
accorded by race, economic power, military force, or station of birth.

We are creating a world where anyone, anywhere may express his or her beliefs,
no matter how singular, without fear of being coerced into silence or
conformity.

Your legal concepts of property, expression, identity, movement, and context
do not apply to us. They are all based on matter, and there is no matter here.

Our identities have no bodies, so, unlike you, we cannot obtain order by
physical coercion. We believe that from ethics, enlightened self-interest, and
the commonweal, our governance will emerge. Our identities may be distributed
across many of your jurisdictions. The only law that all our constituent
cultures would generally recognize is the Golden Rule. We hope we will be able
to build our particular solutions on that basis. But we cannot accept the
solutions you are attempting to impose.

In the United States, you have today created a law, the Telecommunications
Reform Act, which repudiates your own Constitution and insults the dreams of
Jeff`
