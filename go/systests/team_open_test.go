package systests

import (
	"strings"
	"testing"

	"golang.org/x/net/context"

	"github.com/keybase/client/go/libkb"
	keybase1 "github.com/keybase/client/go/protocol/keybase1"
	"github.com/keybase/client/go/teams"
	"github.com/stretchr/testify/require"
)

func TestTeamOpenAutoAddMember(t *testing.T) {
	tt := newTeamTester(t)
	defer tt.cleanup()

	own := tt.addUser("own")
	roo := tt.addUser("roo")

	nameStr, err := libkb.RandString("tt", 5)
	require.NoError(t, err)
	nameStr = strings.ToLower(nameStr)

	cli := own.teamsClient
	createRes, err := cli.TeamCreate(context.TODO(), keybase1.TeamCreateArg{
		Name:                 nameStr,
		SendChatNotification: false,
		Open:                 true,
	})

	_ = createRes
	t.Logf("Open team name is %q", nameStr)

	roo.teamsClient.TeamRequestAccess(context.TODO(), keybase1.TeamRequestAccessArg{Name: nameStr})

	own.kickTeamRekeyd()
	own.waitForTeamChangedGregor(nameStr, keybase1.Seqno(2))

	team, err := teams.Load(context.TODO(), own.tc.G, keybase1.LoadTeamArg{
		Name:        nameStr,
		ForceRepoll: true,
	})
	require.NoError(t, err)

	role, err := team.MemberRole(context.TODO(), roo.userVersion())
	require.NoError(t, err)
	require.Equal(t, role, keybase1.TeamRole_READER)
}

func TestTeamSetOpen(t *testing.T) {
	tt := newTeamTester(t)
	defer tt.cleanup()

	own := tt.addUser("own")

	teamName := own.createTeam()
	t.Logf("Open team name is %q", teamName)

	team, err := teams.Load(context.TODO(), own.tc.G, keybase1.LoadTeamArg{
		Name:        teamName,
		ForceRepoll: true,
	})
	require.NoError(t, err)
	require.Equal(t, team.IsOpen(), false)

	tname, _ := keybase1.TeamNameFromString(teamName)
	err = teams.ChangeTeamSettings(context.TODO(), own.tc.G, tname.ToTeamID(), true)
	require.NoError(t, err)

	team, err = teams.Load(context.TODO(), own.tc.G, keybase1.LoadTeamArg{
		Name:        teamName,
		ForceRepoll: true,
	})
	require.NoError(t, err)
	require.Equal(t, team.IsOpen(), true)
}

func TestOpenSubteamAdd(t *testing.T) {
	tt := newTeamTester(t)
	defer tt.cleanup()

	own := tt.addUser("own")
	roo := tt.addUser("roo")

	// Creating team, subteam, sending open setting, checking if it's set.

	team := own.createTeam()

	parentName, err := keybase1.TeamNameFromString(team)
	require.NoError(t, err)

	subteam, err := teams.CreateSubteam(context.TODO(), own.tc.G, "zzz", parentName)
	require.NoError(t, err)

	t.Logf("Open team name is %q, subteam is %q", team, subteam)

	subteamObj, err := teams.Load(context.TODO(), own.tc.G, keybase1.LoadTeamArg{
		ID:          *subteam,
		ForceRepoll: true,
	})
	require.NoError(t, err)

	err = teams.ChangeTeamSettings(context.TODO(), own.tc.G, subteamObj.ID, true)
	require.NoError(t, err)

	subteamObj, err = teams.Load(context.TODO(), own.tc.G, keybase1.LoadTeamArg{
		ID:          *subteam,
		ForceRepoll: true,
	})
	require.NoError(t, err)
	require.Equal(t, subteamObj.IsOpen(), true)

	// User requesting access
	subteamNameStr := subteamObj.Name().String()
	roo.teamsClient.TeamRequestAccess(context.TODO(), keybase1.TeamRequestAccessArg{Name: subteamNameStr})

	own.kickTeamRekeyd()
	own.waitForTeamChangedGregor(subteamNameStr, keybase1.Seqno(3))

	subteamObj, err = teams.Load(context.TODO(), own.tc.G, keybase1.LoadTeamArg{
		ID:          *subteam,
		ForceRepoll: true,
	})
	require.NoError(t, err)

	role, err := subteamObj.MemberRole(context.TODO(), roo.userVersion())
	require.NoError(t, err)
	require.Equal(t, role, keybase1.TeamRole_READER)
}

func TestTeamOpenMultipleTars(t *testing.T) {
	tt := newTeamTester(t)
	defer tt.cleanup()

	tar1 := tt.addUser("roo1")
	tar2 := tt.addUser("roo2")
	tar3 := tt.addUser("roo3")
	own := tt.addUser("own")

	team := own.createTeam()
	t.Logf("Open team name is %q", team)

	// tar1 and tar2 request access before team is open.
	tar1.teamsClient.TeamRequestAccess(context.TODO(), keybase1.TeamRequestAccessArg{Name: team})
	tar2.teamsClient.TeamRequestAccess(context.TODO(), keybase1.TeamRequestAccessArg{Name: team})

	// Change settings to open
	tname, _ := keybase1.TeamNameFromString(team)
	err := teams.ChangeTeamSettings(context.TODO(), own.tc.G, tname.ToTeamID(), true)
	require.NoError(t, err)

	// tar3 requests, but rekeyd will grab all requests
	tar3.teamsClient.TeamRequestAccess(context.TODO(), keybase1.TeamRequestAccessArg{Name: team})

	own.kickTeamRekeyd()
	own.waitForTeamChangedGregor(team, keybase1.Seqno(3))

	teamObj, err := teams.Load(context.TODO(), own.tc.G, keybase1.LoadTeamArg{
		Name:        team,
		ForceRepoll: true,
	})
	require.NoError(t, err)

	for i := 0; i < 3; i++ {
		role, err := teamObj.MemberRole(context.TODO(), tt.users[i].userVersion())
		require.NoError(t, err)
		require.Equal(t, role, keybase1.TeamRole_READER)
	}
}
