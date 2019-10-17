import { Emoji } from './data';

interface ABCBook {
  [s: string]: string[];
}

interface SecondLevelABCBook {
  [s: string]: ABCBook;
}

const getABCBook = (arr: string[], from: number): ABCBook => {
  const abcBook: ABCBook = {};

  arr.forEach(el => {
    const firstLetter = el[from];

    if (abcBook[firstLetter] === undefined) {
      abcBook[firstLetter] = [];
      abcBook[firstLetter].push(el);
      return;
    }

    abcBook[firstLetter].push(el);
  });

  return abcBook;
};

const getSecondLevelABCBook = (abcBook: ABCBook, from: number): SecondLevelABCBook => {
  const lettersList = Object.keys(abcBook);
  const secondLevelABCBook: SecondLevelABCBook = {};

  lettersList.forEach(letter => {
    secondLevelABCBook[letter] = getABCBook(abcBook[letter], from + 1);
  });

  return secondLevelABCBook;
};

const getSecondLevelABCBookByArray = (arr: string[], from: number): SecondLevelABCBook => {
  return getSecondLevelABCBook(getABCBook(arr, from), from);
};

export const getSplittedEmoji = (emoji: Emoji): SecondLevelABCBook => {
  const emojiList = Object.keys(emoji);
  return getSecondLevelABCBookByArray(emojiList, 1);
};

export const getFirstNEmojiByLetter = (splittedEmoji: SecondLevelABCBook, letter: string, n: number): string[] => {
  const abcBook = splittedEmoji[letter];
  const letters = Object.keys(abcBook);
  const emojiList: string[] = [];

  letters.some(letter => {
    abcBook[letter].some(el => {
      emojiList.push(el);
      return emojiList.length >= n;
    });

    return emojiList.length >= n;
  });

  return emojiList;
};
