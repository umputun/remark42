import { Emoji } from './data';

interface FirstLevelSplittedEmoji {
  [s: string]: string[];
}

interface SecondLevelSplittedEmoji {
  [s: string]: {
    [s: string]: string[];
  };
}

export const firstLevelSplitting = (emoji: Emoji): FirstLevelSplittedEmoji => {
  const emojiList = Object.keys(emoji);
  const splittedEmoji: FirstLevelSplittedEmoji = {};

  emojiList.forEach(emoji => {
    const firsLetter = emoji[1];

    if (splittedEmoji[firsLetter] === undefined) {
      splittedEmoji[firsLetter] = [];
      splittedEmoji[firsLetter].push(emoji);
      return;
    }

    splittedEmoji[firsLetter].push(emoji);
  });

  return splittedEmoji;
};

export const secondLevelSplitting = (splittedEmoji: FirstLevelSplittedEmoji): SecondLevelSplittedEmoji => {
  const lettersList = Object.keys(splittedEmoji);
  const newSplittedEmoji: SecondLevelSplittedEmoji = {};

  lettersList.forEach(letter => {
    const emojiList = splittedEmoji[letter];
    const split: FirstLevelSplittedEmoji = {};

    emojiList.forEach(emoji => {
      const firsLetter = emoji[2];

      if (split[firsLetter] === undefined) {
        split[firsLetter] = [];
        split[firsLetter].push(emoji);
        return;
      }

      split[firsLetter].push(emoji);
    });

    newSplittedEmoji[letter] = splittedEmoji;
  });

  return newSplittedEmoji;
};
