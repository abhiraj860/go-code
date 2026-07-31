import { createSlice, type PayloadAction } from '@reduxjs/toolkit';

/**
 * Seat selection lives in Redux; everything the SERVER owns lives in React
 * Query. That split is the whole reason both libraries are here rather than
 * one:
 *
 *   - which seats this user has clicked is genuinely client state. It has no
 *     server representation until they submit, it must survive navigating
 *     between the seat map and the summary, and several components need it.
 *   - which seats are actually available is server state. It has an owner
 *     elsewhere, it goes stale on its own, and it needs refetching, caching
 *     and invalidation — all of which React Query does and Redux would make
 *     you hand-roll.
 *
 * Putting availability in Redux is the common mistake, and it produces exactly
 * the bug this system exists to avoid: a cached seat state that disagrees with
 * inventory.
 */

/** Real ticketing systems cap party size; it also bounds the hold request. */
export const MAX_SEATS = 10;

export interface SelectionState {
  eventId: string | null;
  seatIds: string[];
  /** Set once a hold succeeds, so checkout can reference it. */
  holdId: string | null;
  holdExpiresAt: string | null;
  error: string | null;
}

const initialState: SelectionState = {
  eventId: null,
  seatIds: [],
  holdId: null,
  holdExpiresAt: null,
  error: null,
};

const selectionSlice = createSlice({
  name: 'selection',
  initialState,
  reducers: {
    /** Switching events discards the previous selection rather than mixing seats across events. */
    eventOpened(state, action: PayloadAction<string>) {
      if (state.eventId !== action.payload) {
        state.eventId = action.payload;
        state.seatIds = [];
        state.holdId = null;
        state.holdExpiresAt = null;
        state.error = null;
      }
    },

    seatToggled(state, action: PayloadAction<string>) {
      const seatId = action.payload;
      const index = state.seatIds.indexOf(seatId);

      if (index >= 0) {
        state.seatIds.splice(index, 1);
        state.error = null;
        return;
      }
      if (state.seatIds.length >= MAX_SEATS) {
        state.error = `You can select at most ${MAX_SEATS} seats.`;
        return;
      }
      state.seatIds.push(seatId);
      state.error = null;
    },

    selectionCleared(state) {
      state.seatIds = [];
      state.error = null;
    },

    holdAcquired(state, action: PayloadAction<{ holdId: string; seatIds: string[]; expiresAt: string }>) {
      state.holdId = action.payload.holdId;
      // Trust the server's list, not the local one: a partial hold means some
      // requested seats were lost to another buyer, and the UI must show what
      // was actually won rather than what was asked for.
      state.seatIds = action.payload.seatIds;
      state.holdExpiresAt = action.payload.expiresAt;
      state.error = null;
    },

    holdFailed(state, action: PayloadAction<string>) {
      state.error = action.payload;
      state.holdId = null;
      state.holdExpiresAt = null;
    },

    /**
     * A seat this user selected was taken by somebody else. Dropped from the
     * selection immediately, because letting them proceed to checkout with a
     * seat that is gone only moves the failure later.
     */
    seatTakenElsewhere(state, action: PayloadAction<string>) {
      const index = state.seatIds.indexOf(action.payload);
      if (index >= 0) {
        state.seatIds.splice(index, 1);
        state.error = 'One of your seats was just taken by someone else.';
      }
    },
  },
});

export const {
  eventOpened,
  seatToggled,
  selectionCleared,
  holdAcquired,
  holdFailed,
  seatTakenElsewhere,
} = selectionSlice.actions;

export default selectionSlice.reducer;
