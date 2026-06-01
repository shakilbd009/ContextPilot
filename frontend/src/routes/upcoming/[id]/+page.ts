// /upcoming/[id] — detail page
export const ssr = false;

export const load = async ({ params }: { params: { id: string } }) => {
  return { meetingId: params.id };
};