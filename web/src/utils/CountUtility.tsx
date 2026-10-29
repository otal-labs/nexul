// What a count says: "1 ticket", "8 tickets", and "No tickets" rather than a bare zero.
export const countLabel = (n: number, one: string, many = `${one}s`) => (n === 0 ? `No ${many}` : `${n} ${n === 1 ? one : many}`);
