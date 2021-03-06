package cmd

import (
	"fmt"
	"os"

	"github.com/fr3dch3n/commit/git"
	"github.com/fr3dch3n/commit/input"
	"github.com/fr3dch3n/commit/utils"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

// Verbose specifies whether debug-logging should be active.
var Verbose bool

// SkipGitAdd runs 'git add -p' beforehand.
var SkipGitAdd bool

// EmptyCommit makes an commit without any chanes.
var EmptyCommit bool

// Message is the commit-message if specified.
var Message string

func init() {
	rootCmd.PersistentFlags().BoolVarP(&Verbose, "verbose", "v", false, "verbose output")
	rootCmd.PersistentFlags().BoolVarP(&SkipGitAdd, "skip-git-add", "s", false, "do not run git add -p beforehand")
	rootCmd.PersistentFlags().BoolVarP(&EmptyCommit, "empty-commit", "e", false, "make an empty commit")
	rootCmd.PersistentFlags().StringVarP(&Message, "message", "m", "", "provide the commit-message")
}

var rootCmd = &cobra.Command{
	Use:   "commit",
	Short: "Easily build up a commit-message that conforms your team-conventions.",
	Run: func(cmd *cobra.Command, args []string) {
		commit()
	},
}

// Execute is the entrypoint for the whole application.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

// CommitConfigPath specifies the file-name for the config in the home-dir.
const CommitConfigPath = ".commit-config"

// StatePath specifies the file-name for the state-file in the home-dir.
const StatePath = ".commit-config.state"

func commit() {
	if Verbose {
		log.SetLevel(log.DebugLevel)
	}

	if x, _ := utils.Exists(".git"); !x {
		panic("not in a git dir, aborting")
	}

	if !git.AreThereChanges()  && !EmptyCommit{
		fmt.Println("There are no changes to add!")
		os.Exit(0)
	}

	homedir := os.Getenv("HOME")
	configPath := homedir + "/" + CommitConfigPath

	commitConfig, err := input.InitCommitConfig(configPath)
	utils.Check(err)
	log.Debug(commitConfig)

	teamMembers, err := input.InitTeamMembersConfig(commitConfig.TeamMembersConfigPath)
	utils.Check(err)
	log.Debug(teamMembers)

	var me input.TeamMember
	me, err = git.GetTeamMemberByAbbreviation(teamMembers, commitConfig.Abbreviation)
	if err != nil && err.Error() == "not-found" {
		newMember, err := git.GetAndSaveNewTeamMember(commitConfig.TeamMembersConfigPath, commitConfig.Abbreviation, teamMembers)
		if err != nil {
			panic(err)
		}
		me = newMember
	} else if err != nil {
		panic(err)
	}

	state, err := input.ReadState(homedir + "/" + StatePath)
	utils.Check(err)
	log.Debug(state)

	if !EmptyCommit {
		if !SkipGitAdd {
			git.Add("-p", "")
		}
		if !git.AnythingStaged() {
			fmt.Println("There are no staged files.")
			os.Exit(0)
		}
	}
	var pair []input.TeamMember
	var scope string

	pair, err = git.GetPair(commitConfig, state.CurrentPair, teamMembers)
	utils.Check(err)

	ctype, err := input.GetCommitType(false)
	utils.Check(err)

	scope, err = input.GetWithDefault("Current scope", state.CurrentScope)
	utils.Check(err)

	err = input.WriteState(homedir+"/"+StatePath, pair, scope)
	utils.Check(err)
	log.Debugf("Pair: %v", pair)
	log.Debug("Story: " + scope)

	var summary string
	if Message == "" {
		summary, err = input.GetNonEmpty("Summary of your commit")
		utils.Check(err)
	} else {
		summary = Message
	}

	log.Debug("Summary: " + summary)
	reviewedSummary := git.ReviewSummary(summary)
	log.Debug("ReviewedSummary: " + reviewedSummary)

	explanation, err := input.GetMultiLineInput("Why did you choose to do that? ")
	utils.Check(err)
	log.Debug("Explanation: " + explanation)

	commitMsg := git.BuildCommitMsg(ctype, scope, pair, reviewedSummary, explanation, me)
	log.Debug("CommitMsg: " + commitMsg)
	if EmptyCommit {
		git.EmptyCommit(commitMsg)
	} else {
		git.Commit(commitMsg)
	}
}
