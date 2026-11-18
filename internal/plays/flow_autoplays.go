package plays

// fixWithAIAutoPlays are Fix with AI's seeded auto plays, off until a workspace switches them on (ADR 0195): a ticket
// entering progress builds on its developer, and so does a backlog ticket whose last blocker cleared, one at a time.
func fixWithAIAutoPlays() []AutoPlayInput {
	progress := StageProgress
	return []AutoPlayInput{
		{
			Moment: MomentTicketEnteredStage, MomentStage: &progress, RunOn: RunOnDeveloper,
			Conditions: Conditions{Match: MatchAll, Groups: []Group{}},
			Priority:   Priority{Rules: []PriorityRule{}, Otherwise: LevelNormal},
		},
		{
			Moment: MomentTicketUnblocked, RunOn: RunOnDeveloper, OneAtATime: true,
			Conditions: Conditions{Match: MatchAll, Groups: []Group{{Match: MatchAll, Rules: []Rule{
				{Field: FieldStage, Op: OpIs, Values: []string{string(StageBacklog)}},
			}}}},
			Priority: Priority{Rules: []PriorityRule{}, Otherwise: LevelNormal},
		},
	}
}

// testWithAIAutoPlays is Test with AI's seeded auto play, off until switched on: a ticket entering testing is tested by
// whoever moved it there, unless it names a tester, who is waited for.
func testWithAIAutoPlays() []AutoPlayInput {
	testingStage := StageTesting
	return []AutoPlayInput{{
		Moment: MomentTicketEnteredStage, MomentStage: &testingStage, RunOn: RunOnCauser,
		Conditions: Conditions{Match: MatchAll, Groups: []Group{{Match: MatchAll, Rules: []Rule{
			{Field: FieldTester, Op: OpUnset},
		}}}},
		Priority: Priority{Rules: []PriorityRule{}, Otherwise: LevelNormal},
	}}
}
