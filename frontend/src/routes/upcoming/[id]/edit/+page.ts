// /upcoming/[id]/edit — edit page
// Ref: evals/e2e/brd-05-manual-upcoming-meeting-creation.md
// Ref: evals/unit/brd-05-manual-upcoming-meeting-creation.md
export const ssr = false;

export const load = async ({ params }) => {
  return { meetingId: params.id };
};