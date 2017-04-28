package bot_reactions

import (
	"bytes"
	"fmt"
	"sort"
	"strings"
	"text/tabwriter"

	"github.com/bwmarrin/discordgo"

	"github.com/adayoung/ada-bot/settings"
)

type BotReaction interface {
	Help() string
	HelpDetail() string
	Reaction(message *discordgo.Message, author *discordgo.Member, mType string) string
}

var _botReactions map[string]map[string][]BotReaction

func init() {
	_botReactions = make(map[string]map[string][]BotReaction)
}

func addReaction(trigger string, mType string, reaction BotReaction) {
	// FIXME: calls to addReaction should be idempotent, let's
	// not add multiple instances of the same reaction to a trigger
	if _botReactions[mType] == nil {
		_botReactions[mType] = make(map[string][]BotReaction)
	}
	_botReactions[mType][trigger] = append(_botReactions[mType][trigger], reaction)
}

func GetReactions(message *discordgo.Message, author *discordgo.Member, mType string) []string {
	var reactions []string
	if _, ok := _botReactions[mType]["*"]; ok { // Run wildcard triggers first
		for _, reaction := range _botReactions[mType]["*"] {
			if author != nil { // FIXME: This can probably be rephrased to something better
				if author.GuildID == "" { // This is useful only for eliza at the moment :joy:
					reactions = append(reactions, reaction.Reaction(message, author, mType))
				} else {
					_ = reaction.Reaction(message, author, mType) // Wildcard triggers should not respond on channels
				}
			} else {
				_ = reaction.Reaction(message, author, mType) // Wildcard triggers should not respond on channels
			}
		}
	}

	if strings.HasPrefix(message.Content, fmt.Sprintf("%s*", settings.Settings.Discord.BotPrefix)) {
		return reactions // Attempted wildcard trigger! Abort abort!
	}

	if !strings.HasPrefix(message.Content, settings.Settings.Discord.BotPrefix) {
		return reactions // The message is irrelevant, bail out with no reactions
	}

	if strings.TrimSpace(strings.ToLower(message.Content)) == fmt.Sprintf("%shelp", settings.Settings.Discord.BotPrefix) {
		reactions = append(reactions, GenHelp(""))
		return reactions
	}

	if strings.HasPrefix(strings.ToLower(message.Content), fmt.Sprintf("%shelp", settings.Settings.Discord.BotPrefix)) {
		helpDetail := strings.TrimSpace(message.Content[len(settings.Settings.Discord.BotPrefix)+4:]) // len("help") -> 4
		reactions = append(reactions, GenHelp(helpDetail))
		return reactions
	}

	for trigger, _reactions := range _botReactions[mType] {
		if strings.HasPrefix(strings.ToLower(message.Content[len(settings.Settings.Discord.BotPrefix):]), strings.ToLower(trigger)) {
			for _, reaction := range _reactions {
				reactions = append(reactions, reaction.Reaction(message, author, mType))
			}
		}
	}

	return reactions
}

func GenHelp(helpDetail string) string {
	w := &tabwriter.Writer{}
	buf := &bytes.Buffer{}

	w.Init(buf, 0, 8, 0, ' ', 0)
	fmt.Fprintf(w, "```\n")

	triggers := []string{}
	for trigger := range _botReactions["CREATE"] {
		if trigger != "*" {
			triggers = append(triggers, trigger)
		}
	}
	sort.Strings(triggers)

	if helpDetail == "" {
		fmt.Fprintf(w, "I have the following commands available:\n")
	} else {
		fmt.Fprintf(w, fmt.Sprintf("%s%s:\n\n", settings.Settings.Discord.BotPrefix, helpDetail))
	}

	for _, trigger := range triggers {
		for _, item := range _botReactions["CREATE"][trigger] {
			if helpDetail == "" {
				fmt.Fprintf(w, "%s%s \t- \t%s\n",
					settings.Settings.Discord.BotPrefix, trigger,
					item.Help(),
				)
			} else {
				if trigger == helpDetail {
					fmt.Fprintf(w, "%s\n", item.HelpDetail())
				}
			}
		}
	}
	fmt.Fprintf(w, "```")

	w.Flush()
	out := buf.String()
	return out
}
