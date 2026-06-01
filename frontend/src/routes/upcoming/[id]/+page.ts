// /upcoming/[id] — detail page
export const ssr = false;

export const load = async ({ params }) => {
  return { meetingId: params.id };
};