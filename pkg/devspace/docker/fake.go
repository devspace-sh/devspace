package docker

import ()

var fakeClient ClientInterface

//SetFakeClient causes NewClient to return the given fake client. !ONLY FOR TESTING!
func SetFakeClient(fake ClientInterface){
	fakeClient = fake
}