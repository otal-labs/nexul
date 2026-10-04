import type { AnswerValue, QuestionItem, QuestionOption } from "@/models/Question";

// Throwaway prototype data: a client's requirements doc, four rounds of questions, and the doc once every answer is in.
export interface ProtoQuestion extends QuestionItem {
  why: string;
}

export interface ProtoRound {
  n: number;
  questions: ProtoQuestion[];
  // The next round's one plain line under this round's "Anything else?".
  reply: string;
}

// The suggested option comes first and says so; nothing is picked for the client.
const opts = (suggested: string, ...rest: (string | QuestionOption)[]): QuestionOption[] => [
  { label: `${suggested} (Suggested)`, value: suggested },
  ...rest.map((o) => (typeof o === "string" ? { label: o } : o)),
];

export const DOC_BEFORE = `## What we want
An app so our customers can book grooming for their dogs online instead of ringing us all day. We have two groomers (Sam and Priya) and one van that does mobile visits on Fridays.

## Booking
- Customer picks a service and a time
- Bigger dogs take longer, maybe?
- Should remind people the day before
- Deposits?? Lots of no-shows in summer

## Services
Full groom, bath and brush, nail trim, puppy intro. Prices depend on size and coat. The van costs extra.

## Our side
We need to see the day's bookings on the tablet at the front desk. Sam wants to block out his lunch.

## Nice to have
Photos of the dog after the groom sent to the owner. Loyalty card thing like the paper one we have now.
`;

export const DOC_AFTER = `## What we want
An app so our customers can book grooming for their dogs online instead of ringing us all day, running before the summer rush in June. We have two groomers, Sam and Priya, and one van that does mobile visits on Fridays. Customers who still prefer to ring keep doing so: the front desk enters their booking on the tablet, into the same diary.

## Booking
- Customer picks a service, a time, and a groomer: Sam, Priya, or whoever is free first.
- How long the appointment takes comes from the service and the dog's size.
- The owner gets a reminder the day before, by text message and by email.

## Services
Full groom, bath and brush, nail trim, puppy intro. Prices depend on size and coat: the app shows a "from" price before booking and the final price is set at the desk. The van costs extra.

## Mobile van
Fridays only, for anyone within 5 miles of the salon.

## Our side
The front desk and both groomers use the app. The tablet at the front desk shows the day as a timeline per groomer. Sam can block out his lunch, and so can Priya. A customer's address and phone number are seen by the front desk and by the groomer on that booking.

## Nice to have
Photos of the dog after the groom sent to the owner. Loyalty works like the paper card: every 10th groom is free.

## Open points
- Whether customers pay a deposit when they book, and what happens when someone cancels late.
`;

export const ROUNDS: ProtoRound[] = [
  {
    n: 1,
    questions: [
      {
        id: "r1q1",
        text: "How should the length of an appointment be worked out?",
        why: "The doc says bigger dogs take longer, maybe.",
        options: opts("From the service and the dog's size", "From the service only, the groomer adjusts it", "One fixed slot for every booking"),
      },
      {
        id: "r1q2",
        text: "Should customers pay a deposit when they book, and when someone cancels late, do they lose it or does the front desk just see a note on their record next time?",
        why: "The doc mentions lots of no-shows in summer.",
        options: opts(
          "Deposit, kept on a cancellation under 24 hours",
          "Deposit, always refunded",
          { label: "No deposit", description: "Late cancellations are noted on the customer's record instead." },
        ),
      },
      {
        id: "r1q3",
        text: "Who can book the van on Fridays?",
        why: "The doc says the van costs extra but not who it is for.",
        options: opts("Anyone in the area it covers", "Existing customers only", "Nobody online, it is booked by phone"),
      },
      {
        id: "r1q4",
        text: "How should the reminder reach the owner?",
        header: "Pick all that apply.",
        why: "The doc asks for a reminder the day before.",
        multi_select: true,
        options: opts(
          "Text message",
          "Email",
          "A notification from an app on their phone, which means every owner installs it first and allows notifications",
        ),
      },
      {
        id: "r1q5",
        text: "Who at the salon will use the app?",
        header: "Pick all that apply.",
        why: "So each person sees what they need and nothing more.",
        multi_select: true,
        options: opts("The front desk", "The groomers", "The owner, for reports"),
      },
    ],
    reply: "Asked as the first question of Round 2.",
  },
  {
    n: 2,
    questions: [
      {
        id: "r2q1",
        text: "Can customers choose which groomer they get?",
        why: "You asked this under Anything else.",
        options: opts("Yes, or whoever is free first", "No, the salon assigns one"),
      },
      {
        id: "r2q2",
        text: "Which area does the van cover?",
        header: "A list of postcodes or a distance from the salon is fine.",
        why: "Anyone in that area can book the van.",
        options: [],
      },
      {
        id: "r2q3",
        text: "What should the tablet at the front desk show for the day?",
        why: "The doc says the front desk needs the day's bookings.",
        options: opts("A timeline per groomer", "One list in time order"),
      },
      {
        id: "r2q4",
        text: "Should customers pay a deposit when they book?",
        why: "It was skipped in Round 1, and no-shows were the doc's main worry.",
        options: opts("Yes, kept on a late cancellation", "Yes, always refunded", "No deposit"),
      },
      {
        id: "r2q5",
        text: "Who should see a customer's address and phone number?",
        why: "The van visits people at home.",
        options: opts("The front desk and the groomer on that booking", "Everyone at the salon", "Only the owner"),
      },
    ],
    reply: "No problem: the front desk enters phone bookings on the tablet, into the same diary.",
  },
  {
    n: 3,
    questions: [
      {
        id: "r3q1",
        text: "Should the photo after the groom go out on its own?",
        why: "The doc lists photos as a nice to have.",
        options: opts("Only when the groomer picks one", "Yes, when the groomer marks the dog done", "Leave photos out for now"),
      },
      {
        id: "r3q2",
        text: "How does the paper loyalty card work today?",
        why: "The doc wants the app to work like it.",
        options: opts("Every 10th groom is free", "A stamp per visit, a free nail trim at 6"),
      },
      {
        id: "r3q3",
        text: "Should prices show before booking, or only once the groomer has seen the dog?",
        why: "Prices depend on size and coat.",
        options: opts("A \"from\" price before, the final one at the desk", "The exact price before booking", "No prices online"),
      },
      {
        id: "r3q4",
        text: "When do you need online booking running?",
        why: "It decides what goes in first.",
        options: opts("Before the summer rush in June", "By the end of the year", "No fixed date"),
      },
    ],
    reply: "Added to the doc under Our side.",
  },
  {
    n: 4,
    questions: [
      {
        id: "r4q1",
        text: "Does the van go out on Saturdays too?",
        why: "You added Saturday mornings to the opening hours.",
        options: opts("No, Fridays only", "Yes, Saturdays as well"),
      },
      {
        id: "r4q2",
        text: "Can new customers book a Saturday?",
        why: "Saturdays are short and fill up fast.",
        options: opts("Yes", "Existing customers only"),
      },
      {
        id: "r4q3",
        text: "How many bookings come in on your busiest day?",
        why: "So the diary and the reminders keep up on a busy Saturday.",
        options: opts("Up to 20", "20 to 40", "More than 40"),
      },
    ],
    reply: "",
  },
];

const pick = (...selected: string[]): AnswerValue => ({ selected });

export const ROUND_1_ANSWERS: Record<string, AnswerValue> = {
  r1q1: pick("From the service and the dog's size"),
  r1q3: pick("Anyone in the area it covers"),
  r1q4: pick("Text message", "Email"),
  r1q5: pick("The front desk", "The groomers"),
};

export const ROUND_2_ANSWERS: Record<string, AnswerValue> = {
  r2q1: pick("Yes, or whoever is free first"),
  r2q2: { text: "5 miles from the salon, but not across the river because of the bridge traffic on Fridays" },
  r2q3: pick("A timeline per groomer"),
  r2q5: pick("The front desk and the groomer on that booking"),
};

export const ROUND_3_ANSWERS: Record<string, AnswerValue> = {
  r3q2: pick("Every 10th groom is free"),
  r3q3: pick("A \"from\" price before, the final one at the desk"),
  r3q4: pick("Before the summer rush in June"),
};

export const NOTE_1 = "Can people pick which groomer they get? Some dogs only like Priya.";
export const NOTE_2 = "Is it a problem if some of our older customers still want to ring up and book?";
