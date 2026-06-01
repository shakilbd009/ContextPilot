import type { PageLoad } from './$types';

export interface Meeting {
	id: string;
	title: string;
	completedAt: string;
	participants: Array<{ displayName: string; email?: string }>;
}

export interface MeetingsResponse {
	meetings: Meeting[];
	total: number;
	page: number;
	limit: number;
}

export const load: PageLoad = async ({ fetch, url }) => {
	const page = Math.max(1, parseInt(url.searchParams.get('page') ?? '1', 10));
	const limit = 20;

	try {
		const res = await fetch(
			`/api/meetings?page=${page}&limit=${limit}`,
			{ credentials: 'include' }
		);

		if (!res.ok) {
			return {
				meetings: [] as Meeting[],
				total: 0,
				page,
				limit,
				hasMore: false,
				loading: false,
				error: `Failed to load meetings (${res.status})`,
			};
		}

		const data: MeetingsResponse = await res.json();
		const hasMore = (page * limit) < data.total;

		return {
			meetings: data.meetings ?? ([] as Meeting[]),
			total: data.total ?? 0,
			page: data.page ?? page,
			limit: data.limit ?? limit,
			hasMore,
			loading: false,
			error: null,
		};
	} catch (err) {
		return {
			meetings: [] as Meeting[],
			total: 0,
			page,
			limit,
			hasMore: false,
			loading: false,
			error: err instanceof Error ? err.message : 'Network error',
		};
	}
};
